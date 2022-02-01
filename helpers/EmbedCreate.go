package functions

import (
	"github.com/bwmarrin/discordgo"
	"os"
	"time"
)

func EmbedCreate(title string, description string, thumbnail string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{&discordgo.MessageEmbedField{
			Name:   os.Getenv("NAME"),
			Value:  description,
			Inline: true,
		},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: thumbnail,
		},
		Color: 6950317,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    os.Getenv("NAME") + " | Bot created by Leki#6796", // please keep it helps me out <3
			IconURL: "https://cdn.discordapp.com/avatars/404064889192185859/a_6beecf530b78f2dbf5c8f4aa58ca9435.gif?size=80",
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Title:     title,
	}
	return embed
}
