"use strict";
// All what is need for ajax.

const auth = extend({
	token: {
		access: null,
		refrsh: null
	},

	signed() {
		return !!this.token.access;
	},
	signin(t) {
		sessionStorage.setItem('token', JSON.stringify(t));
		this.token.access = t.access;
		this.token.refrsh = t.refrsh;
		this.emit('auth', true);
	},
	signout() {
		sessionStorage.removeItem('token');
		this.token.access = null;
		this.token.refrsh = null;
		this.emit('auth', false);
	},
	signload() {
		try {
			// Load from storage
			const t = JSON.parse(sessionStorage.getItem('token'));
			this.token.access = t.access;
			this.token.refrsh = t.refrsh;
			this.emit('auth', true);
		} catch (e) {
			this.token.access = null;
			this.token.refrsh = null;
			this.emit('auth', false);
		}
	}
}, makeeventmodel());

// ajax calls counter
const ajaxcc = extend({}, makeeventmodel());

const ajaxfail = () => {
	showmsgbox(
		"Server unavailable",
		"Server is currently not available, action can not be done now."
	);
};

const fetchajax = (method, url, body) => {
	let resp;
	return fetch(url, {
		method: method,
		headers: {
			'Accept': 'application/json; charset=utf-8',
			'Content-Type': 'application/json; charset=utf-8'
		},
		body: body && JSON.stringify(body)
	}).then(response => {
		resp = response;
		return response.json();
	}).then(data => {
		resp.data = data;
		return resp;
	});
};

// Sends given request with optional JSON object
const sendjson = (xhr, body) => {
	xhr.responseType = "json";

	xhr.setRequestHeader("Accept", "application/json; charset=utf-8");
	if (body) {
		if (body.constructor.name !== 'FormData') {
			xhr.setRequestHeader("Content-Type", "application/json; charset=utf-8");
			body = JSON.stringify(body);
		} else {
			xhr.setRequestHeader("Content-Type", "multipart/form-data");
		}
	}
	xhr.send(body);
};

const sendjsonauth = (xhr, body) => {
	if (auth.token.access) {
		xhr.setRequestHeader("Authorization", "Bearer " + auth.token.access);
	}
	sendjson(xhr, body);
};

const ajaxjsonauth = (method, url, onload, body, silent) => {
	const onerror = () => {
		if (!silent) {
			ajaxcc.emit('ajax', -1);
		}
		ajaxfail();
	};

	if (!silent) {
		ajaxcc.emit('ajax', +1);
	}
	const xhr = new XMLHttpRequest();
	xhr.onload = () => {
		if (xhr.status === 401 && auth.token.refrsh) { // Unauthorized
			const xhr = new XMLHttpRequest();
			xhr.onload = () => {
				if (xhr.status === 200) { // OK
					auth.signin(xhr.response);

					{
						const xhr = new XMLHttpRequest();
						xhr.onload = () => {
							if (xhr.status === 401) { // Unauthorized
								auth.signout(); // unauthorized again, refresh is outdated
							}
							onload(xhr);
							if (!silent) {
								ajaxcc.emit('ajax', -1);
							}
						};
						xhr.onerror = onerror;
						xhr.open(method, url, true);
						sendjsonauth(xhr, body); // 2-nd try
					}
				} else {
					auth.signout();
					if (!silent) {
						ajaxcc.emit('ajax', -1);
					}
				}
			};
			xhr.onerror = onerror;
			xhr.open("POST", "/api/refrsh", true);
			sendjson(xhr, { refrsh: auth.token.refrsh });
		} else {
			onload(xhr);
			if (!silent) {
				ajaxcc.emit('ajax', -1);
			}
		}
	};
	xhr.onerror = onerror;
	xhr.open(method, url, true);
	sendjsonauth(xhr, body); // 1-st try
};

// The End.
