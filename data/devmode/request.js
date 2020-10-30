"use strict";
// All what is need for ajax.

const auth = extend({
	token: {
		access: null,
		refrsh: null
	},
	login: "",

	signed() {
		return !!this.token.access;
	},
	claims() {
		try {
			const p = this.token.access.split('.');
			return JSON.parse(atob(p[1]));
		} catch {
			return null;
		}
	},
	signin(tok, lgn) {
		sessionStorage.setItem('token', JSON.stringify(tok));
		this.token.access = tok.access;
		this.token.refrsh = tok.refrsh;
		if (lgn) {
			sessionStorage.setItem('login', lgn);
			this.login = lgn;
		}
		this.emit('auth', true);
	},
	signout() {
		sessionStorage.removeItem('token');
		this.token.access = null;
		this.token.refrsh = null;
		// login remains unchanged
		this.emit('auth', false);
	},
	signload() {
		try {
			const tok = JSON.parse(sessionStorage.getItem('token'));
			this.token.access = tok.access;
			this.token.refrsh = tok.refrsh;
			this.login = sessionStorage.getItem('login') || "";
			this.emit('auth', true);
		} catch {
			this.token.access = null;
			this.token.refrsh = null;
			this.login = "";
			this.emit('auth', false);
		}
	}
}, makeeventmodel());

// ajax calls counter
const ajaxcc = extend({}, makeeventmodel());

// returns header properties set for ajax calls with json data.
const ajaxheader = (bearer) => {
	const hdr = {
		'Accept': 'application/json;charset=utf-8',
		'Content-Type': 'application/json;charset=utf-8'
	};
	if (bearer && auth.token.access) {
		hdr['Authorization'] = 'Bearer ' + auth.token.access;
	}
	return hdr;
};

const fetchjson = async (method, url, body) => {
	const response = await fetch(url, {
		method: method,
		headers: ajaxheader(false),
		body: body && JSON.stringify(body)
	});
	// returns undefined on empty body or error in json
	try {
		response.data = await response.json();
	} catch {
		response.data = undefined;
	}
	return response;
};

const fetchjsonauth = async (method, url, body) => {
	const response = await fetch(url, { // 1-st try
		method: method,
		headers: ajaxheader(true),
		body: body && JSON.stringify(body)
	});
	if (response.status === 401 && auth.token.refrsh) { // Unauthorized
		const response = await fetchjson("POST", "/api/auth/refrsh", {
			refrsh: auth.token.refrsh
		});
		if (response.ok) {
			auth.signin(response.data);
			const response = fetch(url, { // 2-nd try
				method: method,
				headers: ajaxheader(true),
				body: body && JSON.stringify(body)
			});
			try { // returns undefined on empty body or error in json
				response.data = await response.json();
			} catch {
				response.data = undefined;
			}
			return response;
		}
		return Promise.reject();
	}
	try { // returns undefined on empty body or error in json
		response.data = await response.json();
	} catch {
		response.data = undefined;
	}
	return response;
};

// The End.
