package main

import (
	"encoding/json"
	//"fmt"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"gopkg.in/natefinch/lumberjack.v2"
)

// AppConfig JSON config object
type AppConfig struct {
	Email   string `json:"email"`
	APIKey  string `json:"api_key"`
	Domains []struct {
		Domain string `json:"domain"`
		Host   string `json:"host"`
		Type   string `json:"type"`
	} `json:"domains"`
	IPFetchURLs struct {
		Ipv4 string `json:"ipv4"`
		Ipv6 string `json:"ipv6"`
	} `json:"IPFetchURLs"`
	InitialIP struct {
		Ipv4 string `json:"ipv4"`
		Ipv6 string `json:"ipv6"`
	} `json:"initialIP"`
}

func loadConfig(filePath string) *AppConfig {
	log.Printf("Reading config from '%v'", filePath)

	// configFile, err := ioutil.ReadFile(filePath)
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatalln("Failed to open config file:", err.Error())
	}
	defer configFile.Close()

	//var config AppConfig
	config := new(AppConfig)
	err = json.NewDecoder(configFile).Decode(&config)
	//err = json.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatalln("failed to parse config file: ", err.Error())
	}

	return config
}

func writeConfig(filePath string, config *AppConfig) {
	log.Println("Updating Config File")

	j, jerr := json.MarshalIndent(config, "", "  ")
	if jerr != nil {
		log.Fatalln("Failed to create JSON:", jerr.Error())
	}

	werr := ioutil.WriteFile(filePath, j, 0644)
	if werr != nil {
		log.Fatalln("Failed to write config:", werr.Error())
	}
}

func getPublicIP(config *AppConfig) (v4 string, v6 string) {

	var ipv4 string
	if len(config.IPFetchURLs.Ipv4) > 0 {
		ipv4 = getHTTPString(config.IPFetchURLs.Ipv4, false)

		if net.ParseIP(ipv4).To4() == nil {
			log.Fatalln("Failed to parse IPv4:", ipv4)
		}
	}

	time.Sleep(time.Millisecond * 500)

	var ipv6 string
	if len(config.IPFetchURLs.Ipv6) > 0 {
		ipv6 = getHTTPString(config.IPFetchURLs.Ipv6, true)

		if net.ParseIP(ipv6).To16() == nil || !strings.Contains(ipv6, ":") {
			log.Fatalln("Failed to parse IPv6:", ipv6)
		}
	}

	return ipv4, ipv6
}

func dialTCP4(network, addr string) (net.Conn, error) {
	return net.Dial("tcp4", addr)
}

func dialTCP6(network, addr string) (net.Conn, error) {
	return net.Dial("tcp6", addr)
}

func getHTTPString(url string, useIPv6 bool) string {
	timeout := time.Duration(15 * time.Second)

	var tr *http.Transport
	if useIPv6 {
		tr = &http.Transport{
			Dial: dialTCP6,
		}

		log.Printf("Fetching URL '%v' using IPv6", url)
	} else {
		tr = &http.Transport{
			Dial: dialTCP4,
		}

		log.Printf("Fetching URL '%v' using IPv4", url)
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Fatalln("Failed to get HTML:", err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Failed to get HTML:", err.Error())
	}

	return strings.TrimSpace(string(body[:]))
}

func initLogging() {
	//file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	//if err != nil {
	//	log.Fatalln("Failed to open log file", err)
	//}

	lj := &lumberjack.Logger{
		Filename:   "ddns.log",
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, //days
	}

	log.SetOutput(io.MultiWriter(lj, os.Stdout))
}

func main() {
	forceUpdate := flag.Bool("force", false, "force update")
	configFile := flag.String("config", "config.json", "config file path")

	flag.Parse()

	initLogging()
	log.Println(strings.Repeat("-", 50))
	log.Println("Starting CloudFlare DDNS update")

	config := loadConfig(*configFile)
	// fmt.Println(config)

	// log.Println("API Key: ", config.APIKey)
	log.Println("Fetching IP address")
	ipv4, ipv6 := getPublicIP(config)
	log.Printf("ipv4: '%v', ipv6: '%v'", ipv4, ipv6)

	if config.InitialIP.Ipv4 == ipv4 && config.InitialIP.Ipv6 == ipv6 && !*forceUpdate {
		log.Println("IP address(s) have not changed, skipping update")
		os.Exit(0)
	}

	config.InitialIP.Ipv4 = ipv4
	config.InitialIP.Ipv6 = ipv6

	writeConfig(*configFile, config)

	// Construct a new API object
	api, err := cloudflare.New(config.APIKey, config.Email)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch user details on the account
	//u, err := api.UserDetails()
	//if err != nil {
	//	log.Fatal(err)
	//}
	// Print user details
	// fmt.Println(u)

	// Fetch the zone ID

	for i, v := range config.Domains {
		fullHost := v.Host + "." + v.Domain

		if v.Type == "A" && len(ipv4) < 1 {
			log.Printf("Skipping [A] '%v', no IPv4 address is available", fullHost)
			continue
		} else if v.Type == "AAAA" && len(ipv6) < 1 {
			log.Printf("Skipping [AAAA] '%v', no IPv6 address is available", fullHost)
			continue
		}

		log.Println(">Starting Zone Update<")
		log.Println("Looking for zone:", v.Domain)
		id, err := api.ZoneIDByName(v.Domain) // Assuming example.com exists in your CloudFlare account already
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Zone ID:", id)

		var rr cloudflare.DNSRecord
		rr = cloudflare.DNSRecord{Name: fullHost, Type: v.Type}
		// Fetch zone details
		//zone, err := api.ZoneDetails(id)
		log.Printf("Looking for record: [%v] '%v'", v.Type, fullHost)
		dnsRecords, err := api.DNSRecords(id, rr)
		if err != nil {
			log.Fatal(err)
		}

		if len(dnsRecords) != 1 {
			log.Fatalln("Expected exactly 1 zone records, received:", dnsRecords)
		}

		log.Println("Record ID:", dnsRecords[0].ID)

		if v.Type == "A" {
			rr = cloudflare.DNSRecord{Name: fullHost, Type: v.Type, Content: ipv4}
		} else if v.Type == "AAAA" {
			rr = cloudflare.DNSRecord{Name: fullHost, Type: v.Type, Content: ipv6}
		} else {
			log.Fatalln("Expected A or AAAA record, was:", v.Type)
		}

		log.Println("Updating DNS entry")
		err = api.UpdateDNSRecord(id, dnsRecords[0].ID, rr)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Successfully updated record!")

		// Only sleep if we have more to process
		if i+1 < len(config.Domains) {
			time.Sleep(time.Second * 2)
		}
	}
}
