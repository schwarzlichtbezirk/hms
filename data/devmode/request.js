"use strict";
// All what is need for ajax.

// Authorization shared object, contains access and refresh tokens.
const token = {};

// URI to jump on logout
const signuri = devmode ? "/dev/sign" : "/sign";

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

const ajaxjson = (method, url, onload, body) => {
	app.loadcount++;
	const xhr = new XMLHttpRequest();
	xhr.onload = () => {
		onload(xhr);
		app.loadcount--;
	};
	xhr.onerror = () => {
		showmsgbox(
			"Server unavailable",
			"Server is currently not available, action can not be done now."
		);
		app.loadcount--;
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
					sessionStorage.setItem('token', JSON.stringify(token));

					{
						const xhr = new XMLHttpRequest();
						xhr.onload = () => {
							if (xhr.status === 401) { // Unauthorized
								logout(); // unauthorized again, something wrong on STS
							} else {
								onload(xhr);
							}
						};
						xhr.open(method, url, true);
						sendjsonauth(xhr, body); // 2-nd try
					}
				} else {
					logout();
				}
			};
			xhr.open("POST", "/api/refresh", true);
			sendjson(xhr, { refresh: token.refresh });
		} else {
			onload(xhr);
		}
	};
	xhr.open(method, url, true);
	sendjsonauth(xhr, body); // 1-st try
};

const logout = () => {
	token.access = null;
	token.refresh = null;
	sessionStorage.removeItem('token');
	window.location.assign(signuri); // jump to sign-in page
};

// The End.
