package main

import (
	"fmt"
	"strings"
	"io/ioutil"
	"encoding/json"

	"github.com/nlopes/slack"
	"log"
	"strconv"
	"sort"
)

type User struct {
	Info   slack.User
	Rating int
}

type Token struct {
	Token string   `json:"token"`
}

type Message struct {
	ChannelId string
	Timestamp string
	Payload   string
	Rating    int
	User      User
}

type BotCentral struct {
	Channel *slack.Channel
	Event   *slack.MessageEvent
	UserId  string
}

type AttachmentChannel struct {
	Channel      *slack.Channel
	Attachment   *slack.Attachment
	DisplayTitle string
}

type Messages []Message

func (u Messages) Len() int {
	return len(u)
}
func (u Messages) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}
func (u Messages) Less(i, j int) bool {
	return u[i].Rating > u[j].Rating
}

type ActiveUsers []User

func (u ActiveUsers) GetMeanRating() string {
	var sum float64
	length := u.Len()
	for i := 0; i < length; i++ {
		sum += float64(u[i].Rating)
	}
	return fmt.Sprintf("%6.3f", sum / float64(length))
}

func (u ActiveUsers) FindUser(ID string) User {
	for i := 0; i < u.Len(); i++ {
		if (u[i].Info.ID == ID || u[i].Info.Name == ID || u[i].Info.RealName == ID) {
			return u[i];
		}
	}
	return User{}
}

func (u ActiveUsers) Len() int {
	return len(u)
}
func (u ActiveUsers) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}
func (u ActiveUsers) Less(i, j int) bool {
	return u[i].Rating > u[j].Rating
}

const TOP = 20

var (
	api *slack.Client
	botKey Token
	activeUsers ActiveUsers
	userMessages Messages
	botId string
	botCommandChannel chan *BotCentral
	botReplyChannel chan AttachmentChannel
)

func handleReaction(reaction, userId string, isAdded bool) {
	switch reaction {
	case
		"-1",
		"hankey",
		"disappointed",
		"disappointed_relieved",
		"confused",
		"expressionless",
		"rage",
		"rage1",
		"rage2",
		"rage3",
		"rage4":
		for i, v := range activeUsers {
			if v.Info.ID == userId {
				if (isAdded) {
					activeUsers[i].Rating--
				} else {
					activeUsers[i].Rating++
				}
			}
		}
	default:
		for i, v := range activeUsers {
			if v.Info.ID == userId {
				if isAdded {
					activeUsers[i].Rating++
				} else {
					activeUsers[i].Rating--
				}
			}
		}
	}
}

func init() {
	file, err := ioutil.ReadFile("./token.json")

	if err != nil {
		log.Fatal("File doesn't exist")
	}

	if err := json.Unmarshal(file, &botKey); err != nil {
		log.Fatal("Cannot parse token.json")
	}
}

func handleBotCommands(c chan AttachmentChannel) {
	commands := map[string]string{
		"top":"See the top rank of user rating by a provided number of top spots.",
		"bottom":"See the bottom rank of user rating by a provided number of bottom spots.",
		"help":"See the available bot commands.",
		"mean":"See how the rating of the selected user looks like, comparing to the mean of all users.",
		"mean of":"See how the rating of the selected user looks like, comparing to the mean of all users.",
		"top messages": "See the top ranking messages in the current channel.", }

	var attachmentChannel AttachmentChannel

	for {
		botChannel := <-botCommandChannel
		attachmentChannel.DisplayTitle = "Fortune & Karam with Luck"
		attachmentChannel.Channel = botChannel.Channel
		commandArray := strings.Fields(botChannel.Event.Text)
		switch commandArray[1] {
		case "help":
			fields := make([]slack.AttachmentField, 0)
			for k, v := range commands {
				fields = append(fields, slack.AttachmentField{
					Title: "<bot> " + k,
					Value: v,
				})
			}
			attachment := &slack.Attachment{
				Pretext: "Guru Command List",
				Color: "#B733FF",
				Fields: fields,
			}
			attachmentChannel.Attachment = attachment
			c <- attachmentChannel

		case "top":
			number := commandArray[2]
			if intNumber, err := strconv.Atoi(commandArray[2]); err != nil {
				fmt.Println("Top messages called.")
				sort.Sort(userMessages)
				fields := make([]slack.AttachmentField, 5)
				for i := 0; i < 5; i++ {
					field := slack.AttachmentField{
						Title: fmt.Sprintf("%v %v from %v", userMessages[i].Rating, "Emojis :smile:", userMessages[i].User.Info.RealName),
						Value: userMessages[i].Payload,
						Short: false,
					}
					fields[i] = field
				}

				attachment := &slack.Attachment{
					Pretext: "Top Messages",
					Color: "#0a84c1",
					Fields: fields,
				}
				attachmentChannel.Attachment = attachment
				c <- attachmentChannel

			} else {
				if intNumber > 0 && intNumber <= TOP {
					sort.Sort(activeUsers)
					fields := make([]slack.AttachmentField, intNumber)
					for i := 0; i < intNumber; i++ {
						field := slack.AttachmentField{
							Title: fmt.Sprintf("%v %v", activeUsers[i].Rating, ":star:"),
							Value: fmt.Sprintf("%v", activeUsers[i].Info.RealName),
							Short: false,
						}
						fields[i] = field
					}

					attachment := &slack.Attachment{
						Pretext: "Top " + fmt.Sprintf("%v", number),
						Color: "#36a34f",
						Fields: fields,
					}
					attachmentChannel.Attachment = attachment
					c <- attachmentChannel
				}
			}

		case "bottom":
			bottomNumber := commandArray[2]
			if intNumber, err := strconv.Atoi(bottomNumber); err == nil {
				if intNumber > 0 && intNumber <= 20 {
					sort.Sort(activeUsers)
					fields := make([]slack.AttachmentField, intNumber)
					for i := intNumber - 1; i >= 0; i-- {
						field := slack.AttachmentField{
							Title: fmt.Sprintf("%v %v", activeUsers[len(activeUsers) - 1 - i].Rating, ":star:"),
							Value: fmt.Sprintf("%v", activeUsers[len(activeUsers) - 1 - i].Info.RealName),
							Short: false,
						}
						fields[i] = field
					}

					attachment := &slack.Attachment{
						Pretext: "Bottom " + fmt.Sprintf("%v", intNumber),
						Color: "#b01408",
						Fields: fields,
					}
					attachmentChannel.Attachment = attachment
					c <- attachmentChannel
				}
			}

		case "mean":
			if len(commandArray) > 2 {
				// mean of
				if len(commandArray) == 4 && commandArray[2] == "of" {
					targetUser := activeUsers.FindUser(commandArray[3])

					attachment := &slack.Attachment{
						Pretext: targetUser.Info.RealName,
						Color: "#0a84c1",
						Fields: []slack.AttachmentField{{
							Title: "Score",
							Value: fmt.Sprint(targetUser.Rating),
							Short: true,
						}, {
							Title: "Company Mean Score",
							Value: fmt.Sprint(activeUsers.GetMeanRating()),
							Short: true,
						}},
					}

					attachmentChannel.Channel = botChannel.Channel
					attachmentChannel.Attachment = attachment
					c <- attachmentChannel
				}
			} else {
				// mean
				user := activeUsers.FindUser(botChannel.UserId)
				attachment := &slack.Attachment{
					Pretext: user.Info.RealName,
					Color: "#0a84c1",
					Fields: []slack.AttachmentField{{
						Title: "Score",
						Value: fmt.Sprint(user.Rating),
						Short: true,
					}, {
						Title: "Company Mean Score",
						Value: fmt.Sprint(activeUsers.GetMeanRating()),
						Short: true,
					}},
				}

				attachmentChannel.Attachment = attachment
				c <- attachmentChannel
			}
		}
	}
}

func handleBotReply() {
	for {
		ac := <-botReplyChannel
		params := slack.PostMessageParameters{}
		params.AsUser = true
		params.Attachments = []slack.Attachment{*ac.Attachment}
		_, _, errPostMessage := api.PostMessage(ac.Channel.Name, ac.DisplayTitle, params)
		if errPostMessage != nil {
			log.Fatal(errPostMessage)
		}
	}
}

func main() {
	api = slack.New(botKey.Token)

	rtm := api.NewRTM()

	botCommandChannel = make(chan *BotCentral)
	botReplyChannel = make(chan AttachmentChannel)

	userMessages = make(Messages, 0)

	go rtm.ManageConnection()
	go handleBotCommands(botReplyChannel)
	go handleBotReply()

	Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				botId = ev.Info.User.ID

				for _, u := range ev.Info.Users {
					if u.RealName != "" {
						user := User{
							Info: u,
						}
						activeUsers = append(activeUsers, user)
					}
				}

			// Handle when a new team member joins the domain
			case *slack.TeamJoinEvent:
				user := User{
					Info: ev.User,
				}
				activeUsers = append(activeUsers, user)

			case *slack.MessageEvent:
				channelInfo, err := api.GetChannelInfo(ev.Channel)
				if err != nil {
					log.Fatalln(err)
				}

				botCentral := &BotCentral{
					Channel: channelInfo,
					Event: ev,
					UserId: ev.User,
				}

				user := activeUsers.FindUser(ev.User)

				if ev.Type == "message" && strings.HasPrefix(ev.Text, "<@" + botId + ">") {
					botCommandChannel <- botCentral
				}

				if ev.Type == "message" && ev.User != botId {
					userMessages = append(userMessages, Message{
						User: user,
						ChannelId: ev.Channel,
						Timestamp: ev.Timestamp,
						Payload: ev.Text,
					})
				}

			case *slack.ReactionAddedEvent:

				for i, msg := range userMessages {
					if msg.Timestamp == ev.Item.Timestamp && msg.ChannelId == ev.Item.Channel {
						userMessages[i].Rating++
					}
				}

				handleReaction(ev.Reaction, ev.ItemUser, true)

			case *slack.ReactionRemovedEvent:

				for i, msg := range userMessages {
					if msg.Timestamp == ev.Item.Timestamp && msg.ChannelId == ev.Item.Channel {
						userMessages[i].Rating--
					}
				}

				handleReaction(ev.Reaction, ev.ItemUser, false)

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
			// Ignore other events..
			// fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
