package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/replit/database-go"
	functions "github.com/zLeki/PointBot/helpers"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

type Config struct {
	Name      string
	Errorlog  *log.Logger
	Infolog   *log.Logger
	Debug     bool
	RootPath  string
	Prefix    string
	AdminChan string
}

func init() {
	err := os.Setenv("REPLIT_DB_URL", "https://kv.replit.com/v0/eyJhbGciOiJIUzUxMiIsImlzcyI6ImNvbm1hbiIsImtpZCI6InByb2Q6MSIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjb25tYW4iLCJleHAiOjE2NDM3OTE5MDUsImlhdCI6MTY0MzY4MDMwNSwiZGF0YWJhc2VfaWQiOiJmMDc0NWRhNy1iMGY1LTRhMjYtYjU0Ny03OWE1NWQ1ZDg0NmMifQ.6HcRwjTCnPnKhUUG-rS-QD1n-p3Pf7cKE-75-zIEu3286vdzO3CpcnD1XyQuhWd9WtdiItB4BJD9-L2_NadKZA")
	if err != nil {
		Config{}.Errorlog.Fatalf("Error setting env variable REPLIT_DB_URL")
	}
}

var richestUser = make(map[string]int)
var cache []string

func main() {

	c := Config{}

	c.RootPath, _ = os.Getwd()
	err := godotenv.Load(c.RootPath + "/.env")
	c.Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	c.AdminChan = os.Getenv("OMNICHANNEL")
	c.Prefix = os.Getenv("PREFIX")
	infoLog, errorLog := c.startLoggers()
	c.Errorlog = errorLog
	c.Infolog = infoLog
	if c.Debug {
		c.Infolog.Println("Debug mode enabled")
	}
	if err != nil {
		c.Errorlog.Fatalf("Error loading .env file" + c.RootPath + "/.env")
	}
	dg, err := discordgo.New("Bot " + os.Getenv("BOTTOKEN"))
	if err != nil {
		c.Errorlog.Fatalf("Error creating discord session: %s", err)
	}
	dg.AddHandler(c.messageCreate)
	dg.AddHandler(c.OnReady)
	dg.AddHandler(c.OnReaction)
	err = dg.Open()
	if err != nil {
		c.Errorlog.Fatalf("Error opening connection: %s", err)
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-ch
	defer func(dg *discordgo.Session) {
		err := dg.Close()
		if err != nil {
			c.Errorlog.Fatalf("Error closing connection: %s", err)
		}
	}(dg)
}

func (c *Config) Error(s *discordgo.Session, reason string, ChannelID string) {
	embed, err := s.ChannelMessageSendEmbed(ChannelID, functions.EmbedCreate("Error", reason, "https://i.imgur.com/qs4QOjF.png"))
	if err != nil {
		return
	}
	if c.Debug {
		c.Infolog.Println("Succeeded in sending error message", embed.ID, embed.ChannelID)
	}

}
func (c *Config) OnReady(s *discordgo.Session, event *discordgo.Ready) {
	err := os.Setenv("NAME", s.State.User.Username)
	if err != nil {
		return
	}
	c.Name = s.State.User.Username
	c.Infolog.Println("Bot is ready")
	err = s.UpdateStreamingStatus(0, c.Prefix+"help | "+c.Name, "https://www.twitch.tv/amouranth")
	if err != nil {
		return
	}
}
func (c *Config) checkStat(s *discordgo.Session, userID string, parm1 int, channelID string) (id string, total int) {
	keys, err := database.ListKeys("")
	if c.Debug {
		c.Infolog.Println("Keys:", keys)
	}

	if err != nil {
		c.Error(s, err.Error(), channelID)
		c.Errorlog.Println("Error listing keys: ", err)
		return
	}
	total = 0
	for _, v := range keys {
		if strings.Contains(v, userID) {
			parms, _ := strconv.Atoi(strings.Split(v, ":")[1])
			total += parms
		}
	}
	richestUser[userID] = total
	if c.Debug {
		c.Infolog.Println("Total points: ", total)

	}
	return userID, total
}
func (c *Config) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if s.State.User.ID == m.Author.ID {
		return
	}
	if c.Debug {
		c.Infolog.Println("Message received", m.Content)
	}

	if m.Content == c.Prefix+"ping" {
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, functions.EmbedCreate("Ping", "Pong "+s.HeartbeatLatency().String(), "https://i.imgur.com/v2n7qPs.png"))
		if err != nil {
			c.Error(s, "Error sending message: "+err.Error(), m.ChannelID)
			c.Errorlog.Println("Error sending message: ", err)
			return
		}
	} else if m.Content == c.Prefix+"help" {
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, functions.EmbedCreate("Help", "❓Help\n?request <amount>\n"+c.Prefix+"leaderboard\n"+c.Prefix+"points\n"+c.Prefix+"ping", "https://i.imgur.com/NldSwaZ.png"))
		if err != nil {
			c.Error(s, "Error sending message: "+err.Error(), m.ChannelID)
			c.Errorlog.Println("Error sending message: ", err)
			return
		}
	} else if strings.HasPrefix(m.Content, c.Prefix+"request") {
		if contains(cache, m.Author.ID) {
			c.Error(s, "You have already requested a point", m.ChannelID)
			return
		}
		parms := strings.Split(m.Content, " ")
		if len(parms) < 2 || len(parms) > 3 {
			c.Error(s, "Please specify a valid amount", m.ChannelID)
			return
		}
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, functions.EmbedCreate("Request", "This is a request message, a administrator will review this soon. You will be dm'd when a moderator finishes reviewing", "https://i.imgur.com/NldSwaZ.png"))

		if err != nil {
			c.Error(s, "Error sending message: "+err.Error(), m.ChannelID)
			c.Errorlog.Println("Error sending message: ", err)
			return
		}

		a, err := s.ChannelMessageSend(c.AdminChan, "User, <@!"+m.Author.ID+"> is requesting "+parms[1]+" points.")
		go func() {
			cache = append(cache, m.Author.ID+":"+parms[1])
		}()
		if err != nil {
			c.Error(s, err.Error(), m.ChannelID)
		}
		err = s.MessageReactionAdd(c.AdminChan, a.ID, "✅")
		if err != nil {
			c.Error(s, err.Error(), m.ChannelID)
			c.Errorlog.Println("Error adding reaction: ", err)
			return
		}
		err = s.MessageReactionAdd(c.AdminChan, a.ID, "❎")
		if err != nil {
			c.Error(s, err.Error(), m.ChannelID)
			c.Errorlog.Println("Error adding reaction: ", err)
			return
		}
	} else if m.Content == c.Prefix+"leaderboard" {
		keys, err := database.ListKeys("")
		if err != nil {
			c.Error(s, err.Error(), m.ChannelID)
			c.Errorlog.Println("Error listing keys: ", err)
			return
		}

		for _, v := range keys {

			parms := strings.Split(v, ":")
			vale, _ := strconv.Atoi(parms[1])
			if c.Debug {
				c.Infolog.Println(parms[0], vale, m.ChannelID)
			}
			c.checkStat(s, parms[0], vale, m.ChannelID)

		}
		type user struct {
			ID     string
			Points int
		}
		var users []user
		for k, v := range richestUser {
			users = append(users, user{k, v})
		}
		sort.Slice(users, func(i, j int) bool {
			return users[i].Points > users[j].Points
		})

		if len(users) == 0 {
			c.Error(s, "No users found", m.ChannelID)
			return
		}
		switch len(users) {
		case 1:
			_, err := s.ChannelMessageSendEmbed(m.ChannelID, functions.EmbedCreate("Leaderboard", "1. <@"+users[0].ID+"> - "+strconv.Itoa(users[0].Points), "https://i.imgur.com/NldSwaZ.png"))
			if err != nil {
				c.Error(s, "Error sending message: "+err.Error(), m.ChannelID)
				c.Errorlog.Println("Error sending message: ", err)
				return
			}
		case 2:
			_, err := s.ChannelMessageSendEmbed(m.ChannelID, functions.EmbedCreate("Leaderboard", "1. <@"+users[0].ID+"> - "+strconv.Itoa(users[0].Points)+"\n2. <@"+users[1].ID+"> - "+strconv.Itoa(users[1].Points), "https://i.imgur.com/NldSwaZ.png"))
			if err != nil {
				c.Error(s, "Error sending message: "+err.Error(), m.ChannelID)
				c.Errorlog.Println("Error sending message: ", err)
				return
			}
		case 3:
			_, err := s.ChannelMessageSendEmbed(m.ChannelID, functions.EmbedCreate("Leaderboard", "1. <@"+users[0].ID+"> - "+strconv.Itoa(users[0].Points)+"\n2. <@"+users[1].ID+"> - "+strconv.Itoa(users[1].Points)+"\n3. <@"+users[2].ID+"> - "+strconv.Itoa(users[2].Points), "https://i.imgur.com/NldSwaZ.png"))
			if err != nil {
				c.Error(s, "Error sending message: "+err.Error(), m.ChannelID)
				c.Errorlog.Println("Error sending message: ", err)
				return
			}
		}
		if err != nil {
			c.Error(s, "Error sending message: "+err.Error(), m.ChannelID)
			c.Errorlog.Println("Error sending message: ", err)
			return
		}
		//_ = nil
	} else if m.Content == c.Prefix+"points" {
		_, points := c.checkStat(s, m.Author.ID, 0, m.ChannelID)
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, functions.EmbedCreate("Points", "You have "+strconv.Itoa(points)+" points", "https://i.imgur.com/NldSwaZ.png"))
		if err != nil {
			c.Error(s, "Error sending message: "+err.Error(), m.ChannelID)
			c.Errorlog.Println("Error sending message: ", err)
			return
		}
	}
}
func (c *Config) OnReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}
	if r.ChannelID == c.AdminChan {

		if r.Emoji.Name == "✅" {
			switch len(cache) {
			case 0:
				c.Errorlog.Println("No users in cache")

			}
			for _, v := range cache {

				points := strings.Split(v, ":")
				usrChan, err := s.UserChannelCreate(points[0])
				if err != nil {
					c.Error(s, err.Error(), r.ChannelID)
					c.Errorlog.Println("Error creating user channel: ", err)
					return
				}
				_, err = s.ChannelMessageSend(usrChan.ID, "User, <@!"+r.UserID+"> has approved the request for "+points[1]+" points.")
				err = database.Set(v, points[1])
				if err != nil {
					return
				}
				if err != nil {
					c.Error(s, err.Error(), r.ChannelID)
					c.Errorlog.Println("Error sending message: ", err)
					return
				}
				if c.Debug {
					c.Infolog.Println("Set database")
				}

			}
			cache = nil

		} else if r.Emoji.Name == "❎" {

			for _, v := range cache {
				channel := strings.Split(v, ":")[0]
				usrChan, err := s.UserChannelCreate(channel)
				if err != nil {
					c.Error(s, err.Error(), r.ChannelID)
					c.Errorlog.Println("Error creating user channel: ", err)
					return
				}
				_, err = s.ChannelMessageSend(usrChan.ID, "User, <@!"+r.UserID+"> has denied the request.")
				if err != nil {
					return
				}
				if err != nil {
					c.Error(s, err.Error(), r.ChannelID)
					c.Errorlog.Println("Error sending message: ", err)
					return
				}
				return

			}
			c.Error(s, "No users in cache", r.ChannelID)

		}
	}
}

//Images: https://i.imgur.com/v2n7qPs.png-Ping, https://i.imgur.com/NldSwaZ.png-Info, https://i.imgur.com/qs4QOjF.png-Error
// err := database.Set("key", "value")
//	if err != nil {
//		log.Fatalf("Error: %v", err)
//	}
//
//	log.Println(keys)
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func (c *Config) startLoggers() (*log.Logger, *log.Logger) {
	errorLog := log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLog := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	return infoLog, errorLog
}
