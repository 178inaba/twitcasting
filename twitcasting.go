package twitcasting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

const (
	loginURL  = "https://ssl.twitcasting.tv/indexcaslogin.php"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.2840.71 Safari/537.36"
)

// Client is twitcasting client.
type Client struct {
	httpClient *http.Client

	username string
	password string
}

// Comment is ...
type Comment struct {
	Class string `json:"class"`
}

// NewClient is ...
func NewClient(username, password string) (*Client, error) {
	c := http.DefaultClient
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	c.Jar = jar

	return &Client{
		httpClient: c,
		username:   username,
		password:   password,
	}, err
}

// Auth is ...
func (c *Client) Auth() error {
	param := url.Values{}
	param.Set("username", c.username)
	param.Set("password", c.password)
	param.Set("action", "login")
	req, err := http.NewRequest(http.MethodPost, loginURL, strings.NewReader(param.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Accept-Language", "ja")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	cookieURL, err := url.Parse(loginURL)
	if err != nil {
		return err
	}

	cookies := c.httpClient.Jar.Cookies(cookieURL)
	var existID, existSs bool
	for _, cookie := range cookies {
		switch cookie.Name {
		case "tc_id":
			existID = true
		case "tc_ss":
			existSs = true
		}
	}

	if !existID || !existSs {
		return errors.New("fail login")
	}

	return nil
}

// GetMovieID is ...
func (c *Client) GetMovieID(hostName string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://twitcasting.tv/%s", hostName), nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Accept-Language", "ja")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.Errorf("status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	url, exist := doc.Find("#movietitle a").Attr("href")
	if !exist {
		return 0, errors.New("not broadcasting")
	}

	splitURL := strings.Split(url, "/")
	movieID, err := strconv.Atoi(splitURL[len(splitURL)-1])
	if err != nil {
		return 0, err
	}

	return movieID, nil
}

// PostComment is ...
func (c *Client) PostComment(comment, hostName string, movieID int) error {
	param := url.Values{}
	param.Set("m", fmt.Sprint(movieID))
	param.Set("s", comment)
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("http://twitcasting.tv/%s/userajax.php", hostName), strings.NewReader(param.Encode()))
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("c", "post")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Accept-Language", "ja")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var comments []Comment
	err = json.NewDecoder(resp.Body).Decode(&comments)
	if err != nil {
		return err
	}

	var result bool
	for _, comment := range comments {
		if comment.Class == "you" {
			result = true
			break
		}
	}

	if !result {
		return errors.New("post error")
	}

	return nil
}
