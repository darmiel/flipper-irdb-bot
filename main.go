package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/darmiel/irdb-bot/parser"
	"os"
	"os/signal"
	"syscall"
)

type Config struct {
	Token                  string
	CapturedFilesChannelID string
	DebugChannelID         string
	NewFilesChannelID      string
	WhichPython            string
	LinterRoot             string
	FlipperScriptsRoot     string
}

type messageCreateHandler func(session *discordgo.Session, create *discordgo.MessageCreate)

type Bot struct {
	session        *discordgo.Session
	handlers       map[string]messageCreateHandler
	config         *Config
	linter         *parser.LinterParser
	flipperScripts *parser.FlipperScriptsParser
}

func main() {
	var config Config
	if _, err := toml.DecodeFile("discord.toml", &config); err != nil {
		fmt.Println("error reading config,", err)
		return
	}
	discord, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		fmt.Println("error creating client,", err)
		return
	}
	discord.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent

	// create instance of bot and create message handlers
	bot := &Bot{
		session:  discord,
		handlers: make(map[string]messageCreateHandler),
		config:   &config,
		linter: &parser.LinterParser{
			PythonPath: config.WhichPython,
			LinterRoot: config.LinterRoot,
		},
		flipperScripts: &parser.FlipperScriptsParser{
			PythonPath:         config.WhichPython,
			FlipperScriptsRoot: config.FlipperScriptsRoot,
		},
	}

	// #captured-files
	bot.handlers[config.CapturedFilesChannelID] = bot.messageCreateCapturedFiles
	discord.AddHandler(bot.messageCreate)

	if err = discord.Open(); err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// check file
	/*
		if err = bot.checkFile(
			"710491120903127080",
			"test-name.ir",
			"downloads/1064875662147588147/710491120903127080/test.ir",
		); err != nil {
			fmt.Println("error checking file:", err)
		}
	*/

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("shutting down")
	if err = discord.Close(); err != nil {
		fmt.Println("cannot close client,", err)
		return
	}
}

func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if fun, ok := b.handlers[m.ChannelID]; ok {
		fun(s, m)
	}
}
