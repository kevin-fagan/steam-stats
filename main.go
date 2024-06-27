package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/KevinFagan/steam-stats/steam"
	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

var (
	discordToken = os.Getenv("DISCORD_BOT_TOKEN")
	steamKey     = os.Getenv("STEAM_API_KEY")
	steamClient  = steam.Steam{Key: steamKey}
)

func main() {
	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("error creating Discord session")
		return
	}

	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("error opening connection")
		return
	}

	logrus.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!stats") {
		logrus.WithFields(logrus.Fields{
			"author":  m.Message.Author.Username,
			"channel": m.ChannelID,
			"command": m.Content,
		}).Info("Received command")

		args := strings.Split(m.Content, " ")

		// If the message does not have 3 arguments, send a usage message
		// as it is assumed the user did not provide the correct arguments
		if len(args) != 3 {
			s.ChannelMessageSend(m.ChannelID, "Usage: `!stats <profile|friends|games|bans> <steam-profile-url>`")
			return
		}

		// Retriving detailed information about the player
		player, err := steamClient.PlayerWithDetails(steamClient.ResolveID(args[2]))
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "unable to retrieve user information")
			logrus.WithFields(logrus.Fields{
				"author":  m.Message.Author.Username,
				"channel": m.ChannelID,
				"command": m.Content,
				"error":   err,
			}).Error("unable to retrieve user information")
			return
		}

		// Available commands to retrieve information about the player
		if args[1] == "profile" {
			message := messageProfile(player)
			s.ChannelMessageSendEmbed(m.ChannelID, &message)
			return
		}
		if args[1] == "friends" {
			message := messageFriends(m, steamClient, player)
			s.ChannelMessageSendEmbed(m.ChannelID, &message)
			return
		}
		if args[1] == "games" {
			message := messageGames(m, steamClient, player)
			s.ChannelMessageSendEmbed(m.ChannelID, &message)
			return
		}
		if args[1] == "bans" {
			message := messageBans(player)
			s.ChannelMessageSendEmbed(m.ChannelID, &message)
			return
		}
	}
}

func messageFriends(m *discordgo.MessageCreate, steam steam.Steam, player steam.Player) discordgo.MessageEmbed {
	newestFriend := "-"
	oldestFriend := "-"

	friends, err := steam.Friends(player.SteamID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"author":  m.Message.Author.Username,
			"channel": m.ChannelID,
			"command": m.Content,
			"error":   err,
		}).Error("unable to retrieve friend information")
	} else {
		newest, err := steam.Player(friends.Newest().ID)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"author":  m.Message.Author.Username,
				"channel": m.ChannelID,
				"command": m.Content,
				"error":   err,
			}).Error("unable to retrieve friend information")
		}

		newestFriend = newest.Name
		oldest, err := steam.Player(friends.Oldest().ID)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"author":  m.Message.Author.Username,
				"channel": m.ChannelID,
				"command": m.Content,
				"error":   err,
			}).Error("unable to retrieve friend information")
		}
		oldestFriend = oldest.Name
	}

	embed := &discordgo.MessageEmbed{
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Friend information is dependent upon the user's privacy settings.",
		},
		Color: 0x66c0f4,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: player.AvatarFull,
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name: fmt.Sprintf("%s %s", player.Status(), player.Name),
			URL:  player.ProfileURL,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "# of Friends",
				Value:  strconv.Itoa(friends.Count()),
				Inline: true,
			},
			{
				Name:   "Newest Friend",
				Value:  newestFriend,
				Inline: true,
			},
			{
				Name:   "Oldest Friend",
				Value:  oldestFriend,
				Inline: true,
			},
		},
	}
	return *embed
}

func messageGames(m *discordgo.MessageCreate, steam steam.Steam, player steam.Player) discordgo.MessageEmbed {
	allGames, err := steam.Games(player.SteamID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"author":  m.Message.Author.Username,
			"channel": m.ChannelID,
			"command": m.Content,
			"error":   err,
		}).Error("unable to retrieve game information")
	}

	recentGames, err := steam.RecentGames(player.SteamID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"author":  m.Message.Author.Username,
			"channel": m.ChannelID,
			"command": m.Content,
			"error":   err,
		}).Error("unable to retrieve game information")
	}

	mostPlayed := allGames.MostPlayed().Name
	if mostPlayed == "" {
		mostPlayed = "-"
	}

	embed := &discordgo.MessageEmbed{
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Game information is dependent upon the user's privacy settings.",
		},
		Color: 0x66c0f4,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: player.AvatarFull,
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name: fmt.Sprintf("%s %s", player.Status(), player.Name),
			URL:  player.ProfileURL,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Most Played Game",
				Value:  mostPlayed,
				Inline: true,
			},
			{
				Name:   "Total Playtime",
				Value:  fmt.Sprintf("%dh", allGames.TotalHoursPlayed()),
				Inline: true,
			},
			{
				Name:   "",
				Value:  "",
				Inline: true,
			},
			{
				Name:   "Games Owned",
				Value:  strconv.Itoa(len(allGames.Games)),
				Inline: true,
			},
			{
				Name:   "Games Played",
				Value:  strconv.Itoa(allGames.GamesPlayed()),
				Inline: true,
			},
			{
				Name:   "Games Not Played",
				Value:  strconv.Itoa(allGames.GamesNotPlayed()),
				Inline: true,
			},
			{
				Name:   "Last 2 Week Playtime",
				Value:  fmt.Sprintf("%dh", recentGames.HoursPlayed2Weeks()),
				Inline: true,
			},
			{
				Name:   "Last 2 Week Games Played",
				Value:  strconv.Itoa(len(recentGames.Games)),
				Inline: true,
			},
			{
				Name:   "",
				Value:  "",
				Inline: true,
			},
		},
	}
	return *embed
}

func messageBans(player steam.Player) discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Ban information is dependent upon the user's privacy settings.",
		},
		Color: 0x66c0f4,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: player.AvatarFull,
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name: fmt.Sprintf("%s %s", player.Status(), player.Name),
			URL:  player.ProfileURL,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "VAC Banned",
				Value:  strconv.FormatBool(player.VACBanned),
				Inline: true,
			},
			{
				Name:   "# Of VAC Bans",
				Value:  strconv.Itoa(player.NumOfVacBans),
				Inline: true,
			},
			{
				Name:   "# Of Game Bans",
				Value:  strconv.Itoa(player.NumOfGameBans),
				Inline: true,
			},
			{
				Name:   "Days Since Last Ban",
				Value:  fmt.Sprintf("%dd", player.DaysSinceLastBan),
				Inline: true,
			},
			{
				Name:   "Community Banned",
				Value:  strconv.FormatBool(player.CommunityBanned),
				Inline: true,
			},
			{
				Name:   "Economy Banned",
				Value:  player.EconomyBan,
				Inline: true,
			},
		},
	}
	return *embed
}

func messageProfile(player steam.Player) discordgo.MessageEmbed {
	realName := player.RealName
	if realName == "" {
		realName = "-"
	}
	countryCode := player.CountryCode
	if countryCode == "" {
		countryCode = "-"
	}
	stateCode := player.StateCode
	if stateCode == "" {
		stateCode = "-"
	}

	embed := &discordgo.MessageEmbed{
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Profile information is dependent upon the user's privacy settings.",
		},
		Color: 0x66c0f4,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: player.AvatarFull,
		},
		Author: &discordgo.MessageEmbedAuthor{
			Name: fmt.Sprintf("%s %s", player.Status(), player.Name),
			URL:  player.ProfileURL,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Real Name",
				Value:  realName,
				Inline: true,
			},
			{
				Name:   "Country Code",
				Value:  countryCode,
				Inline: true,
			},
			{
				Name:   "State Code",
				Value:  stateCode,
				Inline: true,
			},
			{
				Name:   "Profile Age",
				Value:  player.ProfileAge(),
				Inline: true,
			},
			{
				Name:   "Last Seen",
				Value:  player.LastSeen(),
				Inline: true,
			},
			{
				Name:   "",
				Value:  "",
				Inline: true,
			},
			{
				Name:   "Level Percentile",
				Value:  strconv.FormatFloat(player.PlayerLevelPercentile, 'f', 2, 64),
				Inline: true,
			},
			{
				Name:   "Level",
				Value:  strconv.Itoa(player.PlayerLevel),
				Inline: true,
			},
			{
				Name:   "Badges",
				Value:  strconv.Itoa(int(len(player.Badges))),
				Inline: true,
			},
		},
	}
	return *embed
}
