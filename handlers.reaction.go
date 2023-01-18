package main

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

func (b *Bot) getStartMessage(threadID string) (*discordgo.Message, *discordgo.Channel, error) {
	// get thread for message
	channel, err := b.session.Channel(threadID)
	if err != nil {
		return nil, nil, err
	}
	if !channel.IsThread() {
		fmt.Println("reacted to a ctl message in non-thread channel")
		return nil, nil, errors.New("not a thread")
	}
	// get original message id by title
	var msgID string
	if spl := strings.Split(channel.Name, "|"); len(spl) >= 2 {
		msgID = strings.TrimSpace(spl[1])
	} else {
		return nil, nil, errors.New("invalid thread name")
	}
	if msgID == "" {
		return nil, nil, errors.New("message id was empty")
	}
	// get message
	origMsg, err := b.session.ChannelMessage(b.config.NewFilesChannelID, msgID)
	return origMsg, channel, err
}

func (b *Bot) messageReact(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	// ignore if reactor was self
	if m.UserID == s.State.User.ID {
		return
	}
	msg, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		fmt.Println("cannot get message for reaction get,", err)
		return
	}
	// ignore if target message not from self
	if msg.Author.ID != s.State.User.ID {
		return
	}
	if strings.HasSuffix(msg.Content, "ctl$") {
		origMsg, channel, err := b.getStartMessage(msg.ChannelID)
		if err != nil {
			fmt.Println("cannot get start message for thread", msg.ChannelID, "::", err)
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
			_, _ = s.ChannelMessageSend(channel.ID, "File was marked as **COMPLETED** by <@"+m.UserID+">")
			b.unsafeEmojiAdd(origMsg.ID, origMsg.ChannelID, "✅")
		case "💩":
			_, _ = s.ChannelMessageSend(channel.ID, "File was marked as **REJECTED** by <@"+m.UserID+">")
			b.unsafeEmojiAdd(origMsg.ID, origMsg.ChannelID, "💩")
		case "🦋":
			b.unsafeEmojiAdd(origMsg.ID, origMsg.ChannelID, "🦋")
		}

		// lock and archive channel
		archived, locked := true, true
		_, _ = b.session.ChannelEdit(msg.ChannelID, &discordgo.ChannelEdit{
			Archived: &archived,
			Locked:   &locked,
		})
	}
}
