package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html/charset"
	"gopkg.in/yaml.v3"
)

type item struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

type config struct {
	Feeds            []feedConfiguration `yaml:"feeds"`
	DiscordAuthToken string              `yaml:"discord_token"`
	DiscordChannel   string              `yaml:"discord_channel"`
	discordChannelId string
}

type feedConfiguration struct {
	Url            string           `yaml:"url"`
	Interval       int              `yaml:"interval"`
	Matches        []string         `yaml:"match"`
	Folder         string           `yaml:"folder"`
	Cookie         string           `yaml:"cookie"`
	matchesRegexen []*regexp.Regexp // Compiled after we read the config
}

func main() {
	localShareFolder := os.Getenv("HOME") + "/.local/share/rssdd"
	localConfigFolder := os.Getenv("HOME") + "/.local/config/rssdd"
	configPath := filepath.Join(localConfigFolder, "config.yaml")
	createEmptyConfigIfNotExists(configPath)
	config, err := readConfig(configPath)
	dbPath := filepath.Join(localShareFolder, "downloads.db")
	fmt.Printf("dbPath: %v\n", dbPath)

	var discord *discordgo.Session
	if config.DiscordAuthToken != "" {
		discord, err = discordgo.New("Bot " + config.DiscordAuthToken)
		if err != nil {
			fmt.Println("error creating Discord session,", err)
			return
		}

		err = discord.Open()
		if err != nil {
			fmt.Println("error opening connection,", err)
			return
		}

		user, err := discord.User("@me")
		if err != nil {
			fmt.Println("error obtaining account details,", err)
			return
		}

		fmt.Println("Logged in as " + user.Username + "#" + user.Discriminator)

		for _, guild := range discord.State.Guilds {
			for _, channel := range guild.Channels {
				if channel.Name == config.DiscordChannel {
					config.discordChannelId = channel.ID
				}
			}
		}
	}

	fmt.Println(config)

	db, err := sql.Open("sqlite3", "file:"+dbPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	createDbPathIfNotExists(localShareFolder)
	err = initializeDatabase(db)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, feedConfig := range config.Feeds {
		cfg := feedConfig
		go downloadFeedLoop(cfg, db, discord, config.discordChannelId)
	}

	select {}
}

func createEmptyConfigIfNotExists(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(configPath), os.ModePerm)
		file, err := os.Create(configPath)
		if err != nil {
			fmt.Println(err)
			return err
		}
		defer file.Close()
		config := config{}
		encoder := yaml.NewEncoder(file)
		encoder.Encode(config)
	}

	return nil
}

func readConfig(configPath string) (config config, err error) {
	file, err := os.Open(configPath)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	for i, feed := range config.Feeds {
		newFeed := feed

		for _, match := range feed.Matches {
			match := match
			regex, err := regexp.Compile(match)
			if err != nil {
				fmt.Printf("Failed to compile regex: %v\n", match)
				return config, err
			}
			newFeed.matchesRegexen = append(newFeed.matchesRegexen, regex)
		}

		config.Feeds[i] = newFeed
	}

	return config, nil
}

func downloadFeedLoop(
	config feedConfiguration,
	db *sql.DB,
	discord *discordgo.Session,
	channelId string,
) {
	for {
		items, err := downloadFeed(config.Url)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, item := range items {
			parsedUrl, err := url.Parse(item.Link)
			if err != nil {
				fmt.Println(err)
				continue
			}
			filename := filepath.Base(parsedUrl.Path)

			alreadyDownloaded, err := checkAlreadyDownloaded(db, item.Link)
			if err != nil {
				fmt.Println(err)
				continue
			}

			if alreadyDownloaded {
				continue
			}

			matches := false
			for _, regex := range config.matchesRegexen {
				if regex.MatchString(item.Title) {
					matches = true
					break
				}
			}

			if !matches {
				continue
			}

			filePath := filepath.Join(config.Folder, filename)
			fmt.Printf("Downloading '%s' to '%s'\n", item.Link, filePath)
			err = downloadFile(config.Cookie, item.Link, filePath)
			if err != nil {
				fmt.Println("Error downloading file:", err)
				continue
			}
			addDownloadedItem(db, item)
			sendDiscordNotification(discord, item, channelId)
		}

		time.Sleep(time.Second * time.Duration(config.Interval))
	}
}

func sendDiscordNotification(discord *discordgo.Session, item item, channelId string) {
	if discord == nil {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       item.Title,
		URL:         item.Link,
		Description: fmt.Sprintf("Downloaded '%s' (%s)", item.Title, item.Link),
	}

	_, err := discord.ChannelMessageSendEmbed(channelId, embed)
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}
}

func addDownloadedItem(db *sql.DB, i item) error {
	sqlStatement := `
    insert into downloads (link, title, created_at) values (?, ?, ?)
    `
	createdAt := time.Now().Unix()
	_, err := db.Exec(sqlStatement, i.Link, i.Title, createdAt)

	return err
}

func checkAlreadyDownloaded(db *sql.DB, s string) (bool, error) {
	sqlStmt := `
    select count(*) from downloads where link = ?
    `
	var count int
	err := db.QueryRow(sqlStmt, s).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func downloadFile(cookie string, url, filePath string) error {
	request, err := http.NewRequest("GET", url, nil)
	request.Header.Set("Cookie", cookie)
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func createDbPathIfNotExists(localShareFolder string) {
	path := filepath.Join(localShareFolder, "rssdd")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}
}

func initializeDatabase(db *sql.DB) error {
	sqlStmt := `
    create table if not exists downloads (
        id integer not null primary key,
        title text,
        link text,
        created_at datetime default current_timestamp
    );
    `
	_, err := db.Exec(sqlStmt)
	if err != nil {
		fmt.Printf("%q: %s\n", err, sqlStmt)
		return err
	}

	return nil
}

func downloadFeed(url string) (items []item, err error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	contentType := response.Header.Get("Content-Type")
	contentReader, err := charset.NewReader(response.Body, contentType)
	decoder := xml.NewDecoder(contentReader)

	for {
		token, _ := decoder.Token()
		if token == nil {
			break
		}

		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Local == "item" {
				var item item
				err = decoder.DecodeElement(&item, &element)
				if err != nil {
					return nil, err
				}
				items = append(items, item)
			}
		}
	}

	return items, nil
}
