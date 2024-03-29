package main

import (
	"encoding/json"
	"errors"
	"fabric-byzantine/server"
	"fabric-byzantine/server/helpers"
	"fabric-byzantine/server/mysql"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

var timerFlag = true

func timerTask() {
	c := time.Tick(time.Duration(helpers.GetAppConf().Conf.TxInterval) * time.Second)
	for {
		<-c
		if timerFlag {
			go server.GetSdkProvider().InvokeCC("peer0.org1.example.com", 0, 0, "mychannel1", "token", "transfer",
				[][]byte{[]byte("fab"), []byte("alice"), []byte("bob"), []byte("10"), []byte("false")})
		}
	}
}

var addr = flag.String("addr", ":8080", "http service address")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

func query(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	user := r.FormValue("user")
	peer := r.FormValue("peer")
	fmt.Println(peer)
	index, _ := strconv.Atoi(peer[9:10])
	if index == 1 {
		if k, err := strconv.Atoi(peer[9:11]); err == nil {
			index = k
		}
	}
	data, err := server.GetSdkProvider().QueryCC(index-1, "mychannel1", "token",
		"balance", [][]byte{[]byte("fab"), []byte(user)})
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(200)
		w.Write(data)
	}
}

func invoke(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	peer := r.FormValue("peer")
	index, _ := strconv.Atoi(peer[9:10])
	if index == 1 {
		if k, err := strconv.Atoi(peer[9:11]); err == nil {
			index = k
		}
	}
	data, txId, err := server.GetSdkProvider().InvokeCC(peer, 1, index-1, "mychannel1", "token", "transfer",
		[][]byte{[]byte("fab"), []byte("alice"), []byte("bob"), []byte("10"), []byte("true")})
	fmt.Println("TxId:", txId)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(200)
		w.Write(data)
	}
}

func muma(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	if r.Method == "OPTIONS" {
		return
	}

	rBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	data, txId, err := server.GetSdkProvider().InvokeCC("peer0.org1.example.com", 0, 0, "mychannel1", "token", "setPeer",
		[][]byte{[]byte("fab"), rBody, []byte("false")})

	fmt.Println("TxId:", txId)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	} else {
		go server.ByzantineNum()

		var peers map[string]string
		if err = json.Unmarshal(rBody, &peers); err == nil {
			for k, v := range peers {
				b, _ := strconv.ParseBool(v)
				if !b {
					mysql.UpdatePeers(k, 1)
				}
			}
		}

		w.WriteHeader(200)
		w.Write(data)
	}
}

func block(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	uid := uuid.NewV4().String()
	fmt.Println(uid)
	ch := make(chan []byte)
	server.BlockChans.Store(uid, ch)

	for {
		//mt, message, err := c.ReadMessage()
		//if err != nil {
		//	log.Println("read:", err)
		//	return
		//}
		//log.Printf("msg type: %d, recv: %s", mt, message)
		select {
		case datas := <-ch:
			log.Println("block ws response:", string(datas))
			err = c.WriteMessage(websocket.TextMessage, datas)
			if err != nil {
				log.Println("block ws response err:", err)
				server.BlockChans.Delete(uid)
				return
			}
		}
	}
}

func blockNumber(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	uid := uuid.NewV4().String()
	fmt.Println(uid)
	ch := make(chan uint64)
	server.BlockNumberChans.Store(uid, ch)

	for {
		select {
		case height := <-ch:
			log.Println("block number ws response:", height)
			err = c.WriteMessage(websocket.TextMessage, []byte(strconv.FormatUint(height, 10)))
			if err != nil {
				log.Println("block number ws response err:", err)
				server.BlockNumberChans.Delete(uid)
				return
			}
		}
	}
}

func blockPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	var err error
	var datas []byte
	defer func() {
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(200)
			w.Write(datas)
		}
	}()
	var pageId, size int
	pageId, err = strconv.Atoi(r.FormValue("id"))
	if err != nil {
		return
	}
	size, err = strconv.Atoi(r.FormValue("size"))
	if err != nil {
		return
	}
	if pageId < 1 || size < 1 {
		err = errors.New("invalid pageId")
		return
	}
	datas, err = mysql.BlockPage(pageId, size)
}

func transaction(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	uid := uuid.NewV4().String()
	fmt.Println(uid)
	ch := make(chan []byte)
	server.TxChans.Store(uid, ch)

	for {
		select {
		case datas := <-ch:
			log.Println("transaction ws response:", string(datas))
			err = c.WriteMessage(websocket.TextMessage, datas)
			if err != nil {
				log.Println("transaction ws response err:", err)
				server.TxChans.Delete(uid)
				return
			}
		}
	}
}

func transactionNumber(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	uid := uuid.NewV4().String()
	fmt.Println(uid)
	ch := make(chan uint64)
	server.TxNumberChans.Store(uid, ch)

	for {
		select {
		case number := <-ch:
			log.Println("transaction number ws response:", number)
			err = c.WriteMessage(websocket.TextMessage, []byte(strconv.FormatUint(number, 10)))
			if err != nil {
				log.Println("transaction number ws response err:", err)
				server.TxNumberChans.Delete(uid)
				return
			}
		}
	}
}

func transactionPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	var err error
	var datas []byte
	defer func() {
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(200)
			w.Write(datas)
		}
	}()
	var pageId, size int
	pageId, err = strconv.Atoi(r.FormValue("id"))
	if err != nil {
		return
	}
	size, err = strconv.Atoi(r.FormValue("size"))
	if err != nil {
		return
	}
	if pageId < 1 || size < 1 {
		err = errors.New("invalid pageId")
		return
	}
	datas, err = mysql.TransactionPage(pageId, size)
}

func peerList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	datas, err := mysql.PeerList()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(200)
		w.Write(datas)
	}
}

func getStatistics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	datas, err := json.Marshal(server.StatisticsTable)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(200)
		w.Write(datas)
	}
}

func timerControl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	timerFlag = !timerFlag
	w.WriteHeader(200)
	w.Write([]byte(strconv.FormatBool(timerFlag)))
}

func controllerState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	w.WriteHeader(200)
	w.Write([]byte(strconv.FormatBool(timerFlag)))
}

func main() {
	defer mysql.CloseDB()

	go server.GetSdkProvider().BlockListener("mychannel1")
	go timerTask()

	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/invoke", invoke)
	http.HandleFunc("/query", query)
	http.HandleFunc("/block/page", blockPage)
	http.HandleFunc("/block/number", blockNumber)
	http.HandleFunc("/block", block)
	http.HandleFunc("/transaction/page", transactionPage)
	http.HandleFunc("/transaction/number", transactionNumber)
	http.HandleFunc("/transaction", transaction)
	http.HandleFunc("/muma", muma)
	http.HandleFunc("/peers", peerList)
	http.HandleFunc("/statistics", getStatistics)
	http.HandleFunc("/controller", timerControl)
	http.HandleFunc("/controllerState", controllerState)
	http.HandleFunc("/", home)
	http.HandleFunc("/echo", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("msg type: %d, recv: %s", mt, message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {

    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;

    var print = function(message) {
        var d = document.createElement("div");
        d.innerHTML = message;
        output.appendChild(d);
    };

    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };

    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };

});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to close the connection. 
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output"></div>
</td></tr></table>
</body>
</html>
`))
