package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"github.com/bitly/go-simplejson"
)

func main() {
	if len(os.Args) == 2 {
		_, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err != nil {
			fmt.Println("Kindly Check The Argument...")
			return
		}

		data, err := json.Marshal(map[string]interface{}{
			"method": "StockAccounts.Check",
			"id":     1,
			"params": []map[string]interface{}{map[string]interface{}{"TradeId": os.Args[1]}},
		})

		if err != nil {
			log.Fatalf("Error in Marshal : %v", err)
		}

		resp, err := http.Post("http://127.0.0.1:4417/rpc", "application/json", strings.NewReader(string(data)))

		if err != nil {
			log.Fatalf("Error in Post: %v", err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			log.Fatalf("Error ReadAll: %v", err)
		}

		newjson, err := simplejson.NewJson(body)

		checkError(err)

		fmt.Println("********************************************")
		fmt.Print("Stocks : ")
		stocks := newjson.Get("result").Get("Stocks")
		fmt.Println(stocks)

		fmt.Print("Current Market Value : ")
		currentMarketValue, _ := newjson.Get("result").Get("CurrentMarketValue").Float64()
		fmt.Print("$")
		fmt.Println(currentMarketValue)

		fmt.Print("Unvested Amount: ")
		unvestedAmount, _ := newjson.Get("result").Get("UnvestedAmount").Float64()
		fmt.Print("$")
		fmt.Println(unvestedAmount)
		fmt.Println("*********************************************")
	} else if len(os.Args) == 3 {
		budget, err := strconv.ParseFloat(os.Args[2], 64)
		if err != nil {
			fmt.Println("Kindly Check The Argument...")
			return
		}

		data, err := json.Marshal(map[string]interface{}{
			"method": "StockAccounts.Buy",
			"id":     2,
			"params": []map[string]interface{}{map[string]interface{}{"StockSymbolAndPercentage": os.Args[1], "Budget": float32(budget)}},
		})

		if err != nil {
			log.Fatalf("Error in Marshal : %v", err)
		}

		resp, err := http.Post("http://127.0.0.1:4417/rpc", "application/json", strings.NewReader(string(data)))

		if err != nil {
			log.Fatalf("Error in Post : %v", err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			log.Fatalf("Error in ReadAll : %v", err)
		}

		newjson, err := simplejson.NewJson(body)

		checkError(err)

		fmt.Println("*********************************************")
		fmt.Print("Trade Id : ")
		tradeId, _ := newjson.Get("result").Get("TradeId").Int()
		fmt.Println(tradeId)

		fmt.Print("Stocks : ")
		stocks := newjson.Get("result").Get("Stocks")
		fmt.Println(*stocks)

		fmt.Print("Unvested Amount : ")
		unvestedAmount, _ := newjson.Get("result").Get("UnvestedAmount").Float64()
		fmt.Print("$")
		fmt.Println(unvestedAmount)
		fmt.Println("*********************************************")
	} else if len(os.Args) > 4 || len(os.Args) < 2 {
		fmt.Println("Kindly Provide Correct Arguments...")
		return
	}   else {
		fmt.Println("Unknown Error ...")
		return
	}

}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal Error : %s\n", err.Error())
		log.Fatal("Error : ", err)
		os.Exit(2)
	}
}


