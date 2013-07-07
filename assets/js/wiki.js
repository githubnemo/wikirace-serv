$(document).ready(function() {
	function getPlayerList() {
		return $("#sidebar #players");
	}

	function newPlayerElement(name) {
		return $('<li data-player="'+name+'">'+name+'(<span class="visits"></span>)</li>');
	}

	function findPlayerElement(name) {
		return getPlayerList().find("li").filter(function() {
			return $(this).data("player") == name;
		});
	}

	function visitHandler(message) {
		var $elem = findPlayerElement(message["PlayerName"]).find(".visits");

		var cur = parseInt($elem.text());

		if (isNaN(cur)) {
			$elem.text("1");
		} else {
			$elem.text(cur + 1);
		}

		$("#dialog").text(message["Message"]).dialog({
			title: "visit"
		});

	}

	function joinHandler(message) {
		getPlayerList().append(newPlayerElement(message["PlayerName"]));
	}

	function leaveHandler(message) {
		findPlayerElement(message["PlayerName"]).remove();
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
