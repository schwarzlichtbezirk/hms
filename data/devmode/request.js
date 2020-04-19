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
