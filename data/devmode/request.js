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

const fetchajax = (method, url, body) => {
	let resp;
	return fetch(url, {
		method: method,
		headers: {
			'Accept': 'application/json;charset=utf-8',
			'Content-Type': 'application/json;charset=utf-8'
		},
		body: body && JSON.stringify(body)
	}).then(response => {
		resp = response;
		// returns undefined on empty body or error in json
		return new Promise((resolve) => response.json()
			.then(json => resolve(json))
			.catch(() => resolve(undefined)));
	}).then(data => {
		resp.data = data;
		return resp;
	});
};

const fetchajaxauth = (method, url, body) => {
	let resp;
	const hdr = {
		'Accept': 'application/json;charset=utf-8',
		'Content-Type': 'application/json;charset=utf-8'
	};
	if (auth.token.access) {
		hdr['Authorization'] = 'Bearer ' + auth.token.access;
	}
	return fetch(url, { // 1-st try
		method: method,
		headers: hdr,
		body: body && JSON.stringify(body)
	}).then(response => {
		if (response.status === 401 && auth.token.refrsh) { // Unauthorized
			return fetchajax("POST", "/api/refrsh", {
				refrsh: auth.token.refrsh
			}).then(response => {
				if (response.ok) {
					auth.signin(response.data);
					const hdr = {
						'Accept': 'application/json;charset=utf-8',
						'Content-Type': 'application/json;charset=utf-8'
					};
					if (auth.token.access) {
						hdr['Authorization'] = 'Bearer ' + auth.token.access;
					}
					return fetch(url, { // 2-nd try
						method: method,
						headers: hdr,
						body: body && JSON.stringify(body)
					}).then(response => {
						resp = response;
						// returns undefined on empty body or error in json
						return new Promise((resolve) => response.json()
							.then(json => resolve(json))
							.catch(() => resolve(undefined)));
					});
				} else {
					return Promise.reject();
				}
			});
		} else {
			resp = response;
			// returns undefined on empty body or error in json
			return new Promise((resolve) => response.json()
				.then(json => resolve(json))
				.catch(() => resolve(undefined)));
		}
	}).then(data => {
		resp.data = data;
		return resp;
	});
};

// The End.
