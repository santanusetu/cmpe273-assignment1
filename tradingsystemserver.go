package main

import (
	"fmt"
	"log"
	"errors"
	"os"
	"strconv"
	"strings"
	"math"
	"io/ioutil"
	"net/http"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/justinas/alice"
	"github.com/bitly/go-simplejson"
	"github.com/bakins/net-http-recover"
)

type StockRequest struct {
	StockSymbolAndPercentage string
	Budget                   float32
}

type StockResponse struct {
	TradeId         	int
	Stocks           	[]string
	UnvestedAmount 		float32
}

type ValidateResponse struct {
	Stocks           	[]string
	CurrentMarketValue 	float32
	UnvestedAmount 		float32
}

type CheckRequest struct {
	TradeId string
}

type StockAccounts struct {
	stockPortfolio map[int](*Portfolio)
}

type Portfolio struct {
	stocks           map[string](*Share)
	unvestedAmount float32
}

type Share struct {
	boughtPrice float32
	shareNum    int
}

var st StockAccounts
var tradeId int

func main() {
	var st = (new(StockAccounts))
	tradeId = 499   	//initializing tradeId -- starts from 500

	router := mux.NewRouter()
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterService(st, "")

	chain := alice.New(
		func(h http.Handler) http.Handler {
			return handlers.CombinedLoggingHandler(os.Stdout, h)
		},
		handlers.CompressHandler,
		func(h http.Handler) http.Handler {
			return recovery.Handler(os.Stderr, h, true)
		})

	router.Handle("/rpc", chain.Then(server))
	log.Fatal(http.ListenAndServe(":4417", server))
}

//Function Enabling Buying of Shares
func (st *StockAccounts) Buy(httpRq *http.Request, rq *StockRequest, rsp *StockResponse) error {
	tradeId++
	rsp.TradeId = tradeId

	if st.stockPortfolio == nil {
		st.stockPortfolio = make(map[int](*Portfolio))
		st.stockPortfolio[tradeId] = new(Portfolio)
		st.stockPortfolio[tradeId].stocks = make(map[string]*Share)
	}

	symbolAndPercentages := strings.Split(rq.StockSymbolAndPercentage, ",")
	newbudget := float32(rq.Budget)
	var spent float32

	for _, stk := range symbolAndPercentages {
		splited := strings.Split(stk, ":")
		stkQuote := splited[0]
		percentage := splited[1]
		strPercentage := strings.TrimSuffix(percentage, "%")
		floatPercentage64, _ := strconv.ParseFloat(strPercentage, 32)
		floatPercentage := float32(floatPercentage64 / 100.00)
		currentPrice := checkQuote(stkQuote)
		shares := int(math.Floor(float64(newbudget * floatPercentage / currentPrice)))
		sharesFloat := float32(shares)
		spent += sharesFloat * currentPrice

		if _, ok := st.stockPortfolio[tradeId]; !ok {
			newPortfolio := new(Portfolio)
			newPortfolio.stocks = make(map[string]*Share)
			st.stockPortfolio[tradeId] = newPortfolio
		}
		if _, ok := st.stockPortfolio[tradeId].stocks[stkQuote]; !ok {
			newShare := new(Share)
			newShare.boughtPrice = currentPrice
			newShare.shareNum = shares
			st.stockPortfolio[tradeId].stocks[stkQuote] = newShare
		} else {
			total := float32(sharesFloat*currentPrice) + float32(st.stockPortfolio[tradeId].stocks[stkQuote].shareNum)*st.stockPortfolio[tradeId].stocks[stkQuote].boughtPrice
			st.stockPortfolio[tradeId].stocks[stkQuote].boughtPrice = total / float32(shares+st.stockPortfolio[tradeId].stocks[stkQuote].shareNum)
			st.stockPortfolio[tradeId].stocks[stkQuote].shareNum += shares
		}

		stockBought := stkQuote + ":" + strconv.Itoa(shares) + ":$" + strconv.FormatFloat(float64(currentPrice), 'f', 2, 32)
		rsp.Stocks = append(rsp.Stocks, stockBought)
	}

	leftOver := newbudget - spent
	rsp.UnvestedAmount = leftOver
	st.stockPortfolio[tradeId].unvestedAmount += leftOver

	return nil
}

//Function Enabling Balance Check
func (st *StockAccounts) Check(httpRq *http.Request, checkRq *CheckRequest, checkResp *ValidateResponse) error {
	if st.stockPortfolio == nil {
		return errors.New("Please Set Up The Account First...")
	}

	tradeId64, err := strconv.ParseInt(checkRq.TradeId, 10, 64)
	if err != nil {
		return errors.New("TradeID is Not Correct... ")
	}
	tradeId := int(tradeId64)

	if pocket, ok := st.stockPortfolio[tradeId]; ok {
		var currentMarketVal float32

		for stockquote, sh := range pocket.stocks {
			currentPrice := checkQuote(stockquote)

			var str string
			if sh.boughtPrice < currentPrice {
				str = "+ $" + strconv.FormatFloat(float64(currentPrice), 'f', 2, 32)
			} else if sh.boughtPrice > currentPrice {
				str = "- $" + strconv.FormatFloat(float64(currentPrice), 'f', 2, 32)
			} else {
				str = " $" + strconv.FormatFloat(float64(currentPrice), 'f', 2, 32)
			}

			entry := stockquote + ":" + strconv.Itoa(sh.shareNum) + ":" + str
			checkResp.Stocks = append(checkResp.Stocks, entry)
			currentMarketVal += float32(sh.shareNum) * currentPrice
		}

		checkResp.UnvestedAmount = pocket.unvestedAmount
		checkResp.CurrentMarketValue = currentMarketVal
	} else {
		return errors.New("Wrong Trade Id Entered... ")
	}
	return nil
}


//Function Checking Quote
func checkQuote(stockName string) float32 {
	URLstart := "https://query.yahooapis.com/v1/public/yql?q=select%20LastTradePriceOnly%20from%20yahoo.finance%0A.quotes%20where%20symbol%20%3D%20%22"
	URLend := "%22%0A%09%09&format=json&env=http%3A%2F%2Fdatatables.org%2Falltables.env"

	resp, err := http.Get(URLstart + stockName + URLend)

	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Fatal("Failed to Retrive Result from URL... ")
	}

	newjson, err := simplejson.NewJson(body)
	if err != nil {
		fmt.Println(err)
	}
	price, _ := newjson.Get("query").Get("results").Get("quote").Get("LastTradePriceOnly").String()
	floatPrice, err := strconv.ParseFloat(price, 32)

	return float32(floatPrice)
}

func checkError(err error) {
	if err != nil {
		log.Fatal("Error : ", err)
	}
}