# leaderboard

Demonstrate the technology for Chat Bot / ChatOps potential business, using Go implementation using the [RTM](https://api.slack.com/rtm) (Real Time Messaging) API by Slack.

### Features

- Track how many points you have based on reactions your peers give to your messages
- Track what messages in the channel are most upvoted to quickly have a recap of the day
- Leaderboard functionality showing the top N or bottom M users based on ranking
- Analytics features such as the mean score of the organization as well as per individual
- Trivial ranking algorithm based on which reaction you get to your message

### Development

- Create a [new bot user integration](https://my.slack.com/services/new/bot) on your Slack
- Create a file `token.json` which follows the format of the `token_sample.json` file provided with the Slack Bot Token
- Then simply run `go run main.go` at the project root and it will read off the token from `token.json`

### Slack Commands

Hitting the leaderboard bot commands in a Slack channel.

![@leder help](http://i.imgur.com/p3ljv2N.png)

### Contributors

- [Chris Ng](https://github.com/chrisrng)
- [Ben Chen](https://github.com/bcVamp)
- Alexey Kozyachiy
