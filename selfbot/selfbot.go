package selfbot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"time"
)

// Selfbot represents all information required to run a selfbot. Multiple selfbots
// can be created with this struct
type Selfbot struct {
	Session     *discordgo.Session
	User        *discordgo.User
	Config      Config
	CommandList CommandList
	Log         *log.Entry
}

func NewSelfbot(config Config, commands CommandList) (Selfbot, error) {
	// Create a new Session
	session, err := discordgo.New(config.Token)
	if err != nil {
		return Selfbot{}, err
	}

	// Open the Session
	if err = session.Open(); err != nil {
		return Selfbot{}, err
	}

	// get local User
	user, err := session.User("@me")
	if err != nil {
		return Selfbot{}, fmt.Errorf("error getting local User: %s", user)
	}

	logger := log.WithFields(log.Fields{"User": user.Username + "#" + user.Discriminator})

	logger.Infof("Started selfbot")

	// create the selfbot struct
	bot := Selfbot{
		Session:     session,
		User:        user,
		Config:      config,
		CommandList: commands,
		Log:         logger,
	}

	// init the handlers
	if err := bot.initHandlers(); err != nil {
		return Selfbot{}, nil
	}

	return bot, nil
}

func (bot *Selfbot) Close() {
	bot.Log.Infoln("Closing Session")
	_ = bot.Session.Close()
}

// sendError sends an error to the current channel and deletes it in 5 seconds
func (bot *Selfbot) sendError(channelID string, err error) {
	// if there is a user error, send an embed with the error
	message, err := bot.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Title:       "Error",
			Description: err.Error(),
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