$(document).ready(function() {
	var src = $("#source");

	$("#print").click(function() {
		var req = new XMLHttpRequest();
		req.open("POST", "/print");
		req.send(src.val());
	});

	var img = $("#preview").get(0);
	var last = "";
	function renderLoop() {
		var current = src.val();
		if (current === last) {
			window.setTimeout(renderLoop, 100);
			return;
		}

		var req = new XMLHttpRequest();
		req.open("POST", "/print?preview=1");
		req.responseType = "blob";
		req.onload = function(event) {
			img.src = URL.createObjectURL(req.response);
			img.classList.remove("hidden");
			img.onload = function() {
				URL.revokeObjectURL(this.src);
				window.setTimeout(renderLoop, 100);
			};
		};
		req.onabort = function() {
			window.setTimeout(renderLoop, 100);
		};
		req.onerror = function() {
			window.setTimeout(renderLoop, 100);
		}
		req.ontimeout = function() {
			window.setTimeout(renderLoop, 100);
		}
		req.send(current);
	}
	renderLoop();
});
