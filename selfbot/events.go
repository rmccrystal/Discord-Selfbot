package selfbot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
	"time"
)

func (bot *Selfbot) initHandlers() error {
	// add the handlers
	bot.Session.AddHandler(bot.onMessageCreate)

	return nil
}

// Called when any User sends a message in any channel
func (bot *Selfbot) onMessageCreate(session *discordgo.Session, ev *discordgo.MessageCreate) {
	if ev.Author.ID == bot.User.ID {
		bot.onSendMessage(ev)
		return
	}
}

// Called whenever the local User sends a message
// TODO: Clean up this function
func (bot *Selfbot) onSendMessage(ev *discordgo.MessageCreate) {
	content := ev.Message.Content

	if strings.HasPrefix(content, bot.Config.Prefix) {
		// remove the prefix from content
		content = content[len(bot.Config.Prefix):]

		// if the first character is space, we don't want to interpret the input as a command
		if content[0] == ' ' {
			return
		}

		// get the command args
		parsed := strings.Split(content, " ")

		command := parsed[0]
		args := parsed[1:]

		bot.Log.Debugf("Received command: %s, with args: %v", command, args)

		// delete the command message
		if err := bot.Session.ChannelMessageDelete(ev.ChannelID, ev.Message.ID); err != nil {
			bot.Log.Errorf("Error deleting command message: %s", err)
		}

		// handle the command
		userError, discordError := bot.CommandList.Run(bot, command, args, ev.Message)

		// if there is an internal error, set the user error to an internal error occurred
		if discordError != nil {
			userError = fmt.Errorf("An internal error occurred: %s", discordError)
			bot.Log.Errorf("Error handling command: %s", discordError)
		}

		if userError != nil {
			bot.Log.Errorf("User error: %s", userError)

			// if there is a user error, send an embed with the error
			message, err := bot.Session.ChannelMessageSendComplex(ev.ChannelID, &discordgo.MessageSend{
				Embed: &discordgo.MessageEmbed{
					Title:       "Error",
					Description: userError.Error(),
					Color:       0xea5455,
				},
			})
			if err != nil {
				bot.Log.Debugf("Error sending error to channel %s", err)
			}

			// delete the error message in 5 seconds
			go func() {
				time.Sleep(5 * time.Second)
				if err := bot.Session.ChannelMessageDelete(message.ChannelID, message.ID); err != nil {
					bot.Log.Errorf("Error deleting error message")
				}
			}()
		}
	}
}