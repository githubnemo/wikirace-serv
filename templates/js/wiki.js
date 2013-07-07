$(document).ready(function() {
  try {

    var sock = new WebSocket("ws://91.97.71.252:8080/client");
    console.log("Websocket - status: " + sock.readyState);
    sock.onopen = function(m) { 
      console.log("CONNECTION opened..." + this.readyState);}
    sock.onmessage = function(m) { 
      console.log(m.data);
      //var msg = jQuery.parseJSON(m.data);

      //$('#log').append('<li>' + msg + ' clicked on ' + msg.Message + '</li>');
    }
    sock.onerror = function(m) {
      console.log("Error occured sending..." + m.data);}
    sock.onclose = function(m) { 
      console.log("Disconnected - status " + this.readyState);}
  } catch(exception) {
    console.log(exception);
  }
});