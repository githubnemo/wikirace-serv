$(document).ready(function() {
	function getPlayerList() {
		return $("#sidebar #players");
	}

	// [[Name1, Visits1, $(playerElement1)], ...]
	function getPlayerListArray() {
		return getPlayerList().find('li').map(function (i, e) {
			var $this = $(this);

			return {
				name: $this.data('player'),
				visits: parseInt($this.find('.visits').text()),
				elem: $this
			};
		});
	}

	function newPlayerElement(name) {
		return $('<li data-player="'+name+'">'+name+' <span class="badge visits"></span></li>');
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

		sortNewPlayerVisits(playerName);
	}

	function pathLength(path) {
		return path === null ? 0 : path.length;
	}

	function indexOf(arr, cmp) {
		for (var i = 0; i < arr.length; i++) {
			if (cmp(arr[i])) {
				return i;
			}
		}
		return -1;
	}

	// http://stackoverflow.com/a/698440/1643939
	function swapNodes(a, b) {
		var aparent = a.parentNode;
		var asibling = a.nextSibling === b ? a : a.nextSibling;
		b.parentNode.insertBefore(a, b);
		aparent.insertBefore(b, asibling);
	}

	// Sorts the new player visit count for player `name` in
	// the player list so that the list of players is sorted by
	// the visit count in ascending order. Also animations.
	//
	// It is vital that the list of players IS ALREADY sorted.
	function sortNewPlayerVisits(name) {
		var playerArray = getPlayerListArray().toArray();

		var currentIndex = indexOf(playerArray, function (e) {
			return e.name == name;
		});

		var newIndex = 0;

		// Find first player with less visits than that of this player
		// that comes after the current index. This entry has to be swapped
		// with the currentIndex.
		for (newIndex = currentIndex; newIndex < playerArray.length; newIndex++) {
			if (playerArray[newIndex].visits < playerArray[currentIndex].visits) {
				break;
			}
		}

		// Nothing to do, all sorted well.
		if (newIndex == playerArray.length) {
			return;
		}

		// Move the element from where it was to the new position in a fancy way.
		var $nameElem = playerArray[currentIndex].elem;
		var $beforeElem = playerArray[newIndex].elem;

		$nameElem.fadeOut(function () {
			$nameElem.insertAfter($beforeElem);
			$nameElem.fadeIn();

			// If player was the first, remove first indicator and give it
			// to the new first.
			if (currentIndex == 0) {
				$nameElem.find(".badge").removeClass("badge-success");
				$beforeElem.find(".badge").addClass("badge-success");
			}

			// Invoke another round of sorting as this might not be the only
			// player that needs swapping.
			sortNewPlayerVisits(name);
		});
	}

	function setPageTitle(text) {
		// remove whitespace markup from wiki
		text = text.replace("_", " ", "g");
		$('#pageTitle').text(text);
	}

	function visitHandler(message) {
		if (message["PlayerName"] === message["RecipientName"]) {
			setPageTitle(message["Player"]["Path"][message["Player"]["Path"].length-1]);
		}

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

	function handleMessage(message) {
		messageHandler[message.Type](message)
	}

	var sock = new WebSocket("ws://"+location.host+"/client");

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
