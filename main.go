package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-gomail/gomail"
)

type Task func() float64

var Config struct {
	Bitcoin struct {
		UpperLimit float64
		LowerLimit float64
	}
	Gold struct {
		UpperLimit float64
		LowerLimit float64
	}
	Email struct {
		From          string
		Authorization string
		Host          string
		Port          int
		MailTo        []string
	}
}

const (
	timeLayout     = "2006-01-02 15:04:05"
	configFilePath = "./xmonitor.toml"
	bitcoin        = "bitcoin"
	bitcoinURL     = "https://3rdparty-apis.coinmarketcap.com/v1/cryptocurrency/widget?id=1&convert=BTC,USD,USD"
	gold           = "gold"
	goldURL        = "https://data-asg.goldprice.org/dbXRates/USD,CNY"
	ozToGrams      = 31.1034768
)

func loadConfig() {
	var err error
	if _, err = toml.DecodeFile(configFilePath, &Config); err != nil {
		fmt.Println("toml fail to parse file :", err)
		os.Exit(-1)
	}
	fmt.Printf("%+v \n", Config)
}

func getBitcoinPrice() (price float64) {
	resp, err := http.Get(bitcoinURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read body error:", err)
		return
	}
	//fmt.Println(string(body))

	var retData struct {
		Data struct {
			One struct {
				Quote struct {
					Usd struct {
						LastUpdated      string  `json:"last_updated"`
						MarketCap        float64 `json:"market_cap"`
						PercentChange1h  float64 `json:"percent_change_1h"`
						PercentChange24h float64 `json:"percent_change_24h"`
						PercentChange7d  float64 `json:"percent_change_7d"`
						Price            float64 `json:"price"`
						Volume24h        float64 `json:"volume_24h"`
					} `json:"USD"`
				} `json:"quote"`
			} `json:"1"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &retData)
	if err != nil {
		fmt.Println("json unmarshal error:", err)
		return
	}

	price = retData.Data.One.Quote.Usd.Price
	fmt.Println(time.Now().Format(timeLayout), " ", "bitcoin price: ", retData.Data.One.Quote.Usd.Price, " USD")
	return
}

func getGoldPrice() (price float64) {
	resp, err := http.Get(goldURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read body error:", err)
		return
	}
	//fmt.Println(string(body))

	var retData struct {
		Items []struct {
			ChgXag   float64 `json:"chgXag"`
			ChgXau   float64 `json:"chgXau"`
			Curr     string  `json:"curr"`
			PcXag    float64 `json:"pcXag"`
			PcXau    float64 `json:"pcXau"`
			XagClose float64 `json:"xagClose"`
			XagPrice float64 `json:"xagPrice"`
			XauClose float64 `json:"xauClose"`
			XauPrice float64 `json:"xauPrice"`
		} `json:"items"`
	}
	err = json.Unmarshal(body, &retData)
	if err != nil {
		fmt.Println("json unmarshal error:", err)
		return
	}

	for _, data := range retData.Items {
		if data.Curr == "CNY" {
			price = data.XauPrice
		}
	}

	price = price / ozToGrams
	fmt.Println(time.Now().Format(timeLayout), " ", "gold price: ", price, " CNY")
	return
}

func sendMail(body string) {
	m := gomail.NewMessage()
	m.SetHeader("From", Config.Email.From)
	m.SetHeader("To", Config.Email.MailTo...)
	m.SetHeader("Subject", "xmonitor report !")

	m.SetBody("text/html", body)

	d := gomail.NewDialer(Config.Email.Host, Config.Email.Port, Config.Email.From, Config.Email.Authorization)
	err := d.DialAndSend(m)
	if err != nil {
		fmt.Println("send email err: ", err)
	} else {
		fmt.Println("send email success")
	}
	return
}

func main() {
	loadConfig()

	var taskMap = make(map[string]Task)
	taskMap[bitcoin] = getBitcoinPrice
	taskMap[gold] = getGoldPrice

	for key, task := range taskMap {
		go func(name string, task Task) {
			sendFlag := false

			//每天零点清零邮件发送标志位
			go func() {
				now := time.Now()
				next := now.Add(time.Hour * 24)
				next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
				t := time.NewTimer(next.Sub(now))
				<-t.C

				fmt.Printf("new day start: %s \n", time.Now().Format(timeLayout))
				sendFlag = false
			}()

			for {
				price := task()
				switch name {
				case bitcoin:
					if price >= Config.Bitcoin.UpperLimit || price <= Config.Bitcoin.LowerLimit {
						if !sendFlag {
							body := fmt.Sprintf("bitcoin price %f is out of range, attention please！", price)
							sendMail(body)
							sendFlag = true
						}
					}

				case gold:
					if price >= Config.Gold.UpperLimit || price <= Config.Gold.LowerLimit {
						if !sendFlag {
							body := fmt.Sprintf("gold price %f is out of range, attention please！", price)
							sendMail(body)
							sendFlag = true
						}
					}

				default:
				}

				time.Sleep(time.Minute * 3)
			}
		}(key, task)
	}

	select {}
}
