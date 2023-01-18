package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

func (b *Bot) messageReact(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	// ignore if reactor was self
	if m.UserID == s.State.User.ID {
		return
	}
	// ignore if target message not from self
	msg, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		fmt.Println("cannot get message for reaction get,", err)
		return
	}
	if msg.Author.ID != s.State.User.ID {
		return
	}
	if strings.HasSuffix(msg.Content, "ctl$") {
		// get thread for message
		channel, err := s.Channel(m.ChannelID)
		if err != nil {
			fmt.Println("cannot get channel for message,", err)
			return
		}
		if !channel.IsThread() {
			fmt.Println("reacted to a ctl message in non-thread channel")
			return
		}
		fmt.Println(channel.Name)
		// get original message id by title
		var msgID string
		if spl := strings.Split(channel.Name, "|"); len(spl) >= 2 {
			msgID = strings.TrimSpace(spl[1])
		} else {
			return
		}
		if msgID == "" {
			return
		}
		// get message
		origMsg, err := s.ChannelMessage(b.config.NewFilesChannelID, msgID)
		if err != nil {
			fmt.Println("cannot get original message from thread,", err)
			return
		}

		// "spoilerize" message
		if !strings.HasPrefix(origMsg.Content, "||") || !strings.HasSuffix(origMsg.Content, "||") {
			_, _ = s.ChannelMessageEdit(origMsg.ChannelID, origMsg.ID, fmt.Sprintf("||%s||", origMsg.Content))
		}

		// remove all previous reactions
		for _, reaction := range origMsg.Reactions {
			b.unsafeEmojiRemove(origMsg.ID, origMsg.ChannelID, reaction.Emoji.Name)
		}

		switch m.Emoji.Name {
		case "✅":
			_, _ = s.ChannelMessageSend(channel.ID, "File was marked as **ACCEPTED** by <@"+m.UserID+">")
			b.unsafeEmojiAdd(origMsg.ID, origMsg.ChannelID, "✅")
		case "💩":
			_, _ = s.ChannelMessageSend(channel.ID, "File was marked as **REJECTED** by <@"+m.UserID+">")
			b.unsafeEmojiAdd(origMsg.ID, origMsg.ChannelID, "💩")
		case "🦋":
			b.unsafeEmojiAdd(origMsg.ID, origMsg.ChannelID, "🦋")
		}
	}
}
