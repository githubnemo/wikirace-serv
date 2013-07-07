$(document).ready(function() {
	function visitHandler(message) {
	}

	function joinHandler(message) {
	}

	function leaveHandler(message) {
	}

	function finishHandler(message) {
	}

	function gameOverHandler(message) {
	}

	function fatalStuffHandler(message) {
	}

	var messageHandler = {
		0: visitHandler,
		1: joinHandler,
		2: leaveHandler,
		3: finishHandler,
		4: gameOverHandler,
		5: fatalStuffHandler,
	};

	function getServerAddress() {
		return location.host;
	}

	function parseMessage(rawMessage) {
		var msg = JSON.parse(rawMessage);

		return msg;
	}

	function handleMessage(message) {
		messageHandler[message.Type](message)
	}

	var sock = new WebSocket("ws://"+getServerAddress()+"/client");

	console.log("Websocket - status: " + sock.readyState);

	sock.onopen = function(m) {
		console.log("CONNECTION opened..." + this.readyState);
	}

	sock.onmessage = function(m) {
		console.log(m.data);

		handleMessage(parseMessage(m.data));
		//$('#log').append('<li>' + msg + ' clicked on ' + msg.Message + '</li>');
	}

	sock.onerror = function(m) {
		console.log("Error occured sending..." + m.data);
	}

	sock.onclose = function(m) {
		console.log("Disconnected - status " + this.readyState);
	}
});
