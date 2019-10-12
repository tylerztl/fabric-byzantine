//package main
//
//import (
//	"fabric-byzantine/server"
//	"fabric-byzantine/server/helpers"
//	"net/http"
//	"time"
//
//	"github.com/gin-gonic/gin"
//)
//

package main

import (
	"fabric-byzantine/server"
	"flag"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func timerTask() {
	c := time.Tick(5 * time.Second)
	for {
		<-c
		go server.GetSdkProvider().InvokeCC("mychannel1", "token", "transfer", [][]byte{[]byte("fab"), []byte("alice"), []byte("bob"), []byte("1"), []byte("true")})
	}
}

//func main() {
//	go server.GetSdkProvider().BlockListener("mychannel1")
//	go timerTask()
//
//	router := gin.Default()
//	router.GET("/query", func(c *gin.Context) {
//		data, err := server.GetSdkProvider().QueryCC("mychannel1", "token", "balance", [][]byte{[]byte("fab"), []byte("alice")})
//		if err != nil {
//			logger.Error("query err: %v", err)
//			c.JSON(http.StatusOK, err)
//		} else {
//			c.JSON(http.StatusOK, data)
//		}
//	})
//
//	router.POST("/invoke", func(c *gin.Context) {
//		message := c.PostForm("message")
//
//		c.JSON(200, gin.H{
//			"status":  "posted",
//			"message": message,
//		})
//	})
//
//	_ = router.Run(":8080")
//}

var addr = flag.String("addr", ":8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

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
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func main() {
	go server.GetSdkProvider().BlockListener("mychannel1")
	go timerTask()

	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		data, err := server.GetSdkProvider().QueryCC("mychannel1", "token", "balance", [][]byte{[]byte("fab"), []byte("alice")})
		if err != nil {
			log.Printf("query err: %v", err)
			w.Write(nil)
		} else {
			w.Write(data)
		}
	})
	http.HandleFunc("/invoke", func(w http.ResponseWriter, r *http.Request) {

	})
	http.HandleFunc("/echo", echo)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
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
