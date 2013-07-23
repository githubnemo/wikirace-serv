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

	function logMessage(message) {
		$('#log').append('<li>' + message + '</li>');
	}

	function getPlayerVisits(playerName) {
		var $elem = findPlayerElement(playerName).find(".visits");

		return parseInt($elem.text());
	}

	function setPlayerVisits(playerName, visits) {
		findPlayerElement(playerName).find(".visits").text(visits);
	}

	function visitHandler(message) {
		console.log(getPlayerVisits(message["PlayerName"]));

		setPlayerVisits(message["PlayerName"], getPlayerVisits(message["PlayerName"]) + 1);
	}

	function joinHandler(message) {
		var player = message["Player"];

		// Player already in list, is no new join.
		if (findPlayerElement(player["Name"]).length > 0) {
			return;
		}

		getPlayerList().append(newPlayerElement(player["Name"]));

		var visits = player["Path"] === null ? 1 : player["Path"].length;

		setPlayerVisits(player["Name"], visits);

		logMessage(player["Name"] + 'has joined game.');
	}

	function leaveHandler(message) {
		findPlayerElement(message["PlayerName"]).remove();
		logMessage(message["PlayerName"] + ' has left the game.');
	}

	function finishHandler(message) {
		$("#dialog").text(message["PlayerName"] + " WINS").dialog({
			title: "visit"
		});
		logMessage(message["PlayerName"] + ' won the game.');
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

		handleMessage(JSON.parse(m.data));
	}

	sock.onerror = function(m) {
		console.log("Error occured sending..." + m.data);
	}

	sock.onclose = function(m) {
		console.log("Disconnected - status " + this.readyState);
	}
});
