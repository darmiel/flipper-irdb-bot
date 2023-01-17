package main

import "fmt"

func (b *Bot) debug(message string) {
	fmt.Println("DEBUG(msg):", message)
	_, err := b.session.ChannelMessageSend(b.config.DebugChannelID, message)
	if err != nil {
		fmt.Println("WARN: cannot send debug message,", err)
	}
}

func (b *Bot) error(message string, errSend error) {
	fmt.Println("DEBUG(err):", message)
	_, err := b.session.ChannelMessageSend(b.config.DebugChannelID, message+": "+errSend.Error())
	if err != nil {
		fmt.Println("WARN: cannot send debug message,", err)
	}
}
