$(document).ready(function() {
  try {
    var sock = new WebSocket("ws://localhost:8080/client");
    console.log("Websocket - status: " + sock.readyState);
    sock.onopen = function(m) { 
      console.log("CONNECTION opened..." + this.readyState);}
    sock.onmessage = function(m) { 
      console.log("received: " + m.data)
    }
    sock.onerror = function(m) {
      console.log("Error occured sending..." + m.data);}
    sock.onclose = function(m) { 
      console.log("Disconnected - status " + this.readyState);}
  } catch(exception) {
    console.log(exception);
  }
});