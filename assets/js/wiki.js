$(document).ready(function() {
	function getPlayerList() {
		return $("#sidebar #players");
	}

	function newPlayerElement(name) {
		return $('<li data-player="'+name+'">'+name+' (<span class="visits"></span>)</li>');
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

	function pathLength(path) {
		return path === null ? 0 : path.length;
	}

	function visitHandler(message) {
		$('#pageTitle').text(message["Player"]["Path"][message["Player"]["Path"].length-1]);

		setPlayerVisits(message["PlayerName"], pathLength(message["Player"]["Path"]));
	}

	function joinHandler(message) {
		var player = message["Player"];

		// Player not in list, add him and print to log.
		if (findPlayerElement(player["Name"]).length == 0) {
			getPlayerList().append(newPlayerElement(player["Name"]));

			logMessage(player["Name"] + 'has joined game.');
		}

		// Update the player's visits in any case (see #4).
		// This prevents a race between the template and the websocket
		// connection.
		setPlayerVisits(player["Name"], pathLength(player["Path"]));
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
