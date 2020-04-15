"use strict";
// All what is need for ajax.

// Authorization shared object, contains access and refresh tokens.
const token = {
	access: null,
	refrsh: null
};

// Sends given request with optional JSON object
const sendjson = (xhr, body) => {
	xhr.responseType = "json";

	xhr.setRequestHeader("Accept", "application/json; charset=utf-8");
	if (body) {
		if (body.constructor.name !== 'FormData') {
			xhr.setRequestHeader("Content-Type", "application/json");
			body = JSON.stringify(body);
		} else {
			xhr.setRequestHeader("Content-Type", "multipart/form-data");
		}
	}
	xhr.send(body);
};

const sendjsonauth = (xhr, body) => {
	xhr.setRequestHeader("Authorization", "Bearer " + token.access);
	sendjson(xhr, body);
};

const ajaxjson = (method, url, onload, body, silent) => {
	if (!silent) {
		app.loadcount++;
	}
	const xhr = new XMLHttpRequest();
	xhr.onload = () => {
		onload(xhr);
		if (!silent) {
			app.loadcount--;
		}
	};
	xhr.onerror = () => {
		if (!silent) {
			app.loadcount--;
		}
		showmsgbox(
			"Server unavailable",
			"Server is currently not available, action can not be done now."
		);
	};
	xhr.open(method, url, true);
	sendjson(xhr, body);
};

const ajaxjsonauth = (method, url, onload, body) => {
	const xhr = new XMLHttpRequest();
	xhr.onload = () => {
		if (xhr.status === 401) { // Unauthorized
			const xhr = new XMLHttpRequest();
			xhr.onload = () => {
				if (xhr.status === 200) { // OK
					token.access = xhr.response.access;
					token.refrsh = xhr.response.refrsh;
					sessionStorage.setItem('token', JSON.stringify(token));
					app.isauth = true;

					{
						const xhr = new XMLHttpRequest();
						xhr.onload = () => {
							if (xhr.status === 401) { // Unauthorized
								logout(); // unauthorized again, refresh is outdated
							}
							onload(xhr);
						};
						xhr.open(method, url, true);
						sendjsonauth(xhr, body); // 2-nd try
					}
				} else {
					logout();
				}
			};
			xhr.open("POST", "/api/refrsh", true);
			sendjson(xhr, { refrsh: token.refrsh });
		} else {
			onload(xhr);
		}
	};
	xhr.open(method, url, true);
	sendjsonauth(xhr, body); // 1-st try
};

const logout = () => {
	token.access = null;
	token.refrsh = null;
	sessionStorage.removeItem('token');
	app.isauth = false;
};

// The End.
