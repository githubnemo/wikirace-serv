try {
          var sock = new WebSocket("ws://localhost:8080/client");
          console.log("Websocket - status: " + sock.readyState);
          sock.onopen = function(m) { 
            console.log("CONNECTION opened..." + this.readyState);}
          sock.onmessage = function(m) { 
            console.log("received: " + m.data)
            $('#chatbox').append('<p>' + m.data + '</p>');}
          sock.onerror = function(m) {
            console.log("Error occured sending..." + m.data);}
          sock.onclose = function(m) { 
            console.log("Disconnected - status " + this.readyState);}
        } catch(exception) {
          console.log(exception);
        }

$('a')
  // check that the pathname component of href doesn't end with "/index.html"
  .filter(function() {
    return !this.href.pathname.match( /\/index\.html$/ );
    // // or you may want to filter out "/index.html" AND "/", e.g.:
    // return !this.href.pathname.match( /\/(index\.html)?$/i )
  }) 
  // add a click event handler that calls LoadPage and prevents following the link
  .click(function(e) {
    e.preventDefault();
    LoadPage(this.href);
  });