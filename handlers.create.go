package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io"
	"math"
	"net/http"
	"os"
	"path"
	"strings"
	"unicode"
)

const MaxDiscordMessageLength = 2000

func (b *Bot) downloadFile(userID, guildID, fileName, url string) (string, error) {
	// pattern:
	// downloads/<guildID>/<userID>/<fileName>
	dir := path.Join("downloads", guildID, userID)
	downloadFileName := path.Join(dir, fileName)
	if stat, err := os.Stat(dir); err != nil {
		// does not exist?
		if !os.IsNotExist(err) {
			b.error("cannot create dir to download file", err)
			return "", err
		}
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			b.error("cannot mkdirAll to download file", err)
			return "", err
		}
	} else if stat != nil {
		// check if stat is dir
		if !stat.IsDir() {
			b.error("dir to download file is not a directory", err)
			return "", err
		}
	}
	// create file
	f, err := os.Create(downloadFileName)
	if err != nil {
		b.error("cannot create download file", err)
		return "", err
	}
	defer f.Close()
	// download file
	resp, err := http.Get(url)
	if err != nil {
		b.error("cannot download file", err)
		return "", err
	}
	defer resp.Body.Close()
	if math.Floor(float64(resp.StatusCode)/100.0) != 2.0 {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}
	if _, err = io.Copy(f, resp.Body); err != nil {
		b.error("cannot copy body from downloaded file", err)
		return "", err
	}
	b.debug("downloaded file: " + downloadFileName)
	return downloadFileName, nil
}

func limit(msg string) string {
	if len(msg) > MaxDiscordMessageLength {
		return msg[:MaxDiscordMessageLength-4] + "..."
	}
	return msg
}

func createMessageLink(guildID, channelID, messageID string) string {
	return fmt.Sprintf("<https://discord.com/channels/%s/%s/%s>",
		guildID, channelID, messageID)
}

func (b *Bot) sendFile(filePath, fileName string, sourceMsg *discordgo.Message) (*discordgo.Message, error) {
	// open file and send to channel
	f, err := os.Open(filePath)
	if err != nil {
		b.error("cannot open file after check", err)
		return nil, err
	}
	defer f.Close()
	data := &discordgo.MessageSend{
		Content: fmt.Sprintf("`%s` by <@%s>\n> %s",
			fileName, sourceMsg.Author.ID, createMessageLink(sourceMsg.GuildID, sourceMsg.ChannelID, sourceMsg.ID)),
		Files: []*discordgo.File{
			{Name: fileName, Reader: f},
		},
	}
	msg, err := b.session.ChannelMessageSendComplex(b.config.NewFilesChannelID, data)
	if err != nil {
		b.error("cannot send new attachment to channel", err)
		return nil, err
	}
	return msg, nil
}

func (b *Bot) getDescription(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		b.error("cannot open file to get description", err)
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	var bob strings.Builder
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "#") {
			bob.WriteString(strings.TrimSpace(line[1:]))
			bob.WriteRune('\n')
		} else if bob.Len() > 0 {
			break
		}
	}
	return bob.String(), nil
}

func createCredits(msg *discordgo.Message) string {
	text := fmt.Sprintf("# Uploaded by %s#%s\n# %s",
		msg.Author.Username, msg.Author.Discriminator, createMessageLink(msg.GuildID, msg.ChannelID, msg.ID))
	// remove any non-ascii characters
	return strings.Map(func(r rune) rune {
		if r <= unicode.MaxLatin1 {
			return r
		}
		return -1
	}, text)
}

func (b *Bot) checkFile(fileName, filePath string, author *discordgo.User, sourceMsg *discordgo.Message) error {
	msg, err := b.sendFile(filePath, fileName, sourceMsg)
	if err != nil {
		return err
	}

	// create thread for message
	thread, err := b.session.MessageThreadStart(msg.ChannelID, msg.ID,
		fmt.Sprintf("%s | %s", fileName, msg.ID),
		10080) // 48 hours
	if err != nil {
		b.error("cannot create thread", err)
		return err
	}

	// send file description
	desc, err := b.getDescription(filePath)
	if err != nil {
		b.error("cannot read description", err)
	} else if desc != "" {
		_, _ = b.session.ChannelMessageSend(thread.ID,
			fmt.Sprintf("**📝 File Description:**\n```\n%s\n```\n*Credits:*\n```\n%s\n```", desc,
				createCredits(sourceMsg)))
	} else {
		_, _ = b.session.ChannelMessageSend(thread.ID,
			"**📝 File Description:**\n*No description found. Consider adding one.*")
	}

	// linter header
	_, _ = b.session.ChannelMessageSend(thread.ID,
		"https://media.discordapp.net/attachments/800114653820747796/1065045437297467432/banner-linter.png?width=2268&height=310")

	// loading message for linter and dupe check
	linterMsg, err := b.session.ChannelMessageSend(thread.ID, "Working on it <a:loading:1064886986780983356>")
	if err != nil {
		b.error("cannot send linter loading message", err)
		return err
	}

	// dupe header
	_, _ = b.session.ChannelMessageSend(thread.ID,
		"https://media.discordapp.net/attachments/800114653820747796/1065045436597018664/banner-dupe-check.png?width=2268&height=310")

	dupeMsg, err := b.session.ChannelMessageSend(thread.ID, "Working on it <a:loading:1064886986780983356>")
	if err != nil {
		b.error("cannot send dupe loading message", err)
		return err
	}

	// control header
	_, _ = b.session.ChannelMessageSend(thread.ID,
		"https://media.discordapp.net/attachments/800114653820747796/1065046081668403311/banner-control.png?width=2268&height=310")

	controlMsg, _ := b.session.ChannelMessageSend(thread.ID,
		"Waiting for checks <a:loading:1064886986780983356>")

	/// run checks
	allGood := true

	// run linter
	linterIssues, linterResult, err := b.linter.Pretty(filePath)
	if err != nil {
		_, _ = b.session.ChannelMessageEdit(thread.ID, linterMsg.ID,
			fmt.Sprintf("💥 Cannot check with linter: %v", err))
		b.unsafeEmojiAdd(msg.ID, msg.ChannelID, "💥")
	} else if len(linterIssues) > 0 {
		_, _ = b.session.ChannelMessageEdit(thread.ID, linterMsg.ID,
			limit(linterResult))
		b.unsafeEmojiAdd(msg.ID, msg.ChannelID, "⚠️")
		allGood = false
	} else {
		_, _ = b.session.ChannelMessageEdit(thread.ID, linterMsg.ID,
			"✅ No Issues Found")
	}

	// run duplicate checker
	dupeDupes, dupeResult, err := b.flipperScripts.PrettyDupeCheck(filePath)
	if err != nil {
		_, _ = b.session.ChannelMessageEdit(thread.ID, dupeMsg.ID,
			fmt.Sprintf("💥 Cannot check with dupe checker: %v", err))
		b.unsafeEmojiAdd(msg.ID, msg.ChannelID, "💥")
	} else if len(dupeDupes) > 0 {
		_, _ = b.session.ChannelMessageEdit(thread.ID, dupeMsg.ID,
			limit(dupeResult))
		b.unsafeEmojiAdd(msg.ID, msg.ChannelID, "‼️")
		allGood = false
	} else {
		_, _ = b.session.ChannelMessageEdit(thread.ID, dupeMsg.ID,
			"✅ No Duplicates Found")
	}

	if allGood {
		b.unsafeEmojiAdd(msg.ID, msg.ChannelID, "👍")
	}

	if controlMsg != nil {
		_, _ = b.session.ChannelMessageEdit(controlMsg.ChannelID, controlMsg.ID,
			"` ✅ ` - Mark File as Completed\n` 💩 ` - Mark File as Rejected\n` 🦋 `️ - Close without Comment\nctl$")

		b.unsafeEmojiAdd(controlMsg.ID, controlMsg.ChannelID, "✅")
		b.unsafeEmojiAdd(controlMsg.ID, controlMsg.ChannelID, "💩")
		b.unsafeEmojiAdd(controlMsg.ID, controlMsg.ChannelID, "🦋")
	}

	return nil
}

func (b *Bot) unsafeEmojiAdd(messageID, channelID, emojiID string) {
	if err := b.session.MessageReactionAdd(channelID, messageID, emojiID); err != nil {
		fmt.Println("cannot add reaction to", messageID, "@", channelID, ",", err)
	}
}

func (b *Bot) unsafeEmojiRemove(messageID, channelID, emojiID string) {
	if err := b.session.MessageReactionRemove(channelID, messageID, emojiID, b.session.State.User.ID); err != nil {
		fmt.Println("cannot remove reaction", emojiID, "from", messageID, "@", channelID, ",", err)
	}
}

func (b *Bot) messageCreateCapturedFiles(_ *discordgo.Session, m *discordgo.MessageCreate) {
	// ignore message without attachments, since we only want to check .ir files
	if len(m.Attachments) == 0 {
		return
	}
	for _, attachment := range m.Attachments {
		if !strings.HasSuffix(attachment.Filename, ".ir") {
			continue
		}
		// add eyes emoji to indicate working
		b.unsafeEmojiAdd(m.ID, m.ChannelID, "👀")
		filePath, err := b.downloadFile(m.Author.ID, m.GuildID, attachment.Filename, attachment.URL)
		if err != nil {
			b.unsafeEmojiRemove(m.ID, m.ChannelID, "👀")
			b.unsafeEmojiAdd(m.ID, m.ChannelID, "💀")
			fmt.Println("skipping attachment", attachment.Filename, "::", attachment.ID)
			continue
		}
		if err = b.checkFile(attachment.Filename, filePath, m.Author, m.Message); err != nil {
			b.unsafeEmojiRemove(m.ID, m.ChannelID, "👀")
			b.unsafeEmojiAdd(m.ID, m.ChannelID, "💀")
			fmt.Println("skipping checking for", attachment.Filename, "::", attachment.ID)
			continue
		}
		b.unsafeEmojiRemove(m.ID, m.ChannelID, "👀")
		b.unsafeEmojiAdd(m.ID, m.ChannelID, "🥳")
	}
}
