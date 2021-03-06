package commands

import (
	"fmt"
	"github.com/Maldris/mathparse"
	"github.com/bwmarrin/discordgo"
	"github.com/common-nighthawk/go-figure"
	"github.com/rmccrystal/discord-selfbot/selfbot"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

func clearPinsCommand(bot *selfbot.Selfbot, args []string, message *discordgo.Message) (userError, discordError error) {
	messages, err := bot.Session.ChannelMessagesPinned(message.ChannelID)
	if err != nil {
		return nil, err
	}

	messagesToDelete := 50
	if len(args) >= 1 {
		messagesToDelete, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("%s is not a valid number", args[0]), nil
		}
	}

	unpinnedMessages := 0
	// iterate over messages in reverse order
	for i := len(messages); i > 0; i-- {
		i := i - 1

		if unpinnedMessages >= messagesToDelete {
			break
		}

		if err := bot.Session.ChannelMessageUnpin(messages[i].ChannelID, messages[i].ID); err != nil {
			return nil, err
		}
		// cache the pins in case we want to restore them
		bot.RemovedPins = append(bot.RemovedPins, *messages[i])

		bot.Log.WithFields(logrus.Fields{
			"content":   messages[i].Content,
			"messageID": messages[i].ID,
			"channelID": messages[i].ChannelID,
		}).Debugf("Unpinned message")
		unpinnedMessages++
	}

	if err := bot.SendInfo(message.ChannelID, fmt.Sprintf("Unpinned %d messages", unpinnedMessages), true); err != nil {
		return nil, err
	}

	return nil, nil
}

func restorePinsCommand(bot *selfbot.Selfbot, _ []string, message *discordgo.Message) (userError, discordError error) {
	var messages []discordgo.Message
	for _, msg := range bot.RemovedPins {
		if msg.ChannelID == message.ChannelID {
			messages = append(messages, msg)
		}
	}

	if len(messages) == 0 {
		return fmt.Errorf("There are no cached unpinned messages for this channel"), nil
	}

	restoredPinsCount := 0

	for _, msg := range messages {
		if err := bot.Session.ChannelMessagePin(msg.ChannelID, msg.ID); err != nil {
			return nil, err
		}

		// Remove the message from bot.RemovedPins
		for idx, pin := range bot.RemovedPins {
			if pin.ID == msg.ID {
				bot.RemovedPins = append(bot.RemovedPins[:idx], bot.RemovedPins[idx+1:]...)
				break
			}
		}

		restoredPinsCount++

		bot.Log.WithFields(logrus.Fields{
			"content":   msg.Content,
			"messageID": msg.ID,
			"channelID": msg.ChannelID,
		}).Debugf("Restored pinned message")
	}

	if err := bot.SendInfo(message.ChannelID, fmt.Sprintf("Restored %d pins", restoredPinsCount), true); err != nil {
		return nil, err
	}

	return nil, nil
}

func clearCommand(bot *selfbot.Selfbot, args []string, message *discordgo.Message) (userError, discordError error) {
	seperatorText := "***"
	var totalLines int
	if len(args) > 0 {
		totalLines, _ = strconv.Atoi(args[0])
	}
	// default to 60 totalLines
	if totalLines == 0 {
		totalLines = 60
	}

	var maxLinesPerMessage = 2000 - (len(seperatorText) * 2)

	sentLines := 0
	// send messages in chunks so we can send over 2000 lines
	for sentLines < totalLines {
		// how many lines we're sending this iteration
		lines := totalLines - sentLines

		// clamp lines
		if lines > maxLinesPerMessage {
			lines = maxLinesPerMessage
		}

		msg := seperatorText + strings.Repeat("\n", lines) + seperatorText
		_, discordError = bot.Session.ChannelMessageSend(
			message.ChannelID,
			msg,
		)
		if discordError != nil {
			return
		}

		sentLines += lines
	}
	return
}

func reactCommand(bot *selfbot.Selfbot, args []string, message *discordgo.Message) (userError, discordError error) {
	// get latest message
	messageHistory, err := bot.Session.ChannelMessages(message.ChannelID, 100, "", "", "")
	if err != nil {
		return nil, err
	}

	if len(messageHistory) == 0 {
		return fmt.Errorf("no messages found"), nil
	}

	reactMessage := messageHistory[0]

	reactText := strings.Join(args, " ")
	var reactEmojis []string
	var usedEmojis []string
	for _, char := range reactText {
		if emojiArr, ok := EmojiDict[char]; ok {
			// find an emoji that hasn't been used
			for _, emoji := range emojiArr {
				// check if the emoji was used already
				used := false
				for _, usedEmoji := range usedEmojis {
					if usedEmoji == emoji {
						used = true
						break
					}
				}
				// if it wasn't used append it to the emoji arrays
				if !used {
					reactEmojis = append(reactEmojis, emoji)
					usedEmojis = append(usedEmojis, emoji)
					break
				}
				// otherwise, continue the loop and check the other emojis
			}
		}
	}

	// react with the emojis
	for _, emoji := range reactEmojis {
		if err := bot.Session.MessageReactionAdd(reactMessage.ChannelID, reactMessage.ID, emoji); err != nil {
			bot.Log.Errorf("error adding reaction")
		}
	}

	return nil, nil
}

func asciiCommand(bot *selfbot.Selfbot, args []string, message *discordgo.Message) (userError, discordError error) {
	input := strings.Join(args, " ")

	// create a figure
	fig := figure.NewFigure(input, bot.Config.DefaultAsciiFont, false)

	// format the text as monospaced
	messageText := fmt.Sprintf("```%s```", fig.String())

	if _, err := bot.Session.ChannelMessageSend(message.ChannelID, messageText); err != nil {
		return nil, err
	}

	return nil, nil
}

// Deletes the past n messages from the User
func deleteCommand(bot *selfbot.Selfbot, args []string, message *discordgo.Message) (userError, discordError error) {
	// if there are no args count is 0, else parse args[0]
	var count int
	if len(args) == 0 {
		count = 1
	} else {
		var err error
		count, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("%s is not a valid number", args[0]), nil
		}
	}

	// delete the command if it's not already deleted
	_ = bot.Session.ChannelMessageDelete(message.ChannelID, message.ID)

	// delete `count` past messages
	deleteCount := 0
	lastDeletedMessageID := ""
	for deleteCount < count {
		messages, err := bot.Session.ChannelMessages(message.ChannelID, 100, lastDeletedMessageID, "", "")
		if err != nil {
			return nil, fmt.Errorf("error getting message history: %s", err)
		}

		for _, message := range messages {
			if deleteCount >= count {
				break
			}

			if message.Author.ID != bot.User.ID {
				continue
			}

			err = bot.Session.ChannelMessageDelete(message.ChannelID, message.ID)
			if err != nil {
				bot.Log.WithFields(logrus.Fields{
					"deleteCount":    deleteCount,
					"messageType":    message.Type,
					"messageContent": message.Content,
				}).Warnf("error deleting message %s: %s", message.Content, err)
				continue
			}
			lastDeletedMessageID = message.ID
			deleteCount += 1
		}
	}

	return nil, nil
}

func calcCommand(bot *selfbot.Selfbot, args []string, message *discordgo.Message) (userError, err error) {
	equation := strings.Join(args, " ")
	parser := mathparse.NewParser(equation)
	parser.Resolve()
	result := fmt.Sprintf("%s = %s", equation, parser.GetExpressionResult())

	if err := bot.SendInfo(message.ChannelID, fmt.Sprintf("`%s`", result), true); err != nil {
		return nil, err
	}
	return nil, nil
}
