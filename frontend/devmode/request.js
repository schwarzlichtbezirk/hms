"use strict";
// All what is need for ajax.

const eventHub = new Vue();

const auth = {
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
		eventHub.$emit('auth', true);
	},
	signout() {
		sessionStorage.removeItem('token');
		this.token.access = null;
		this.token.refrsh = null;
		// login remains unchanged
		eventHub.$emit('auth', false);
	},
	signload() {
		try {
			const tok = JSON.parse(sessionStorage.getItem('token'));
			this.token.access = tok.access;
			this.token.refrsh = tok.refrsh;
			this.login = sessionStorage.getItem('login') || "";
			eventHub.$emit('auth', true);
		} catch {
			this.token.access = null;
			this.token.refrsh = null;
			this.login = "";
			eventHub.$emit('auth', false);
		}
	}
};

// error on HTTP response with given status.
class HttpError extends Error {
	constructor(status, errajax) {
		super(errajax.what);
		this.name = 'HttpError';
		this.status = status;
		extend(this, errajax);
	}
}

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

// make ajax-call with json data.
const fetchjson = async (method, url, body) => {
	return await fetch(url, {
		method: method,
		headers: ajaxheader(false),
		body: JSON.stringify(body)
	});
};

// make ajax-call with json data and get json-response.
const fetchajax = async (method, url, body) => {
	const response = await fetch(url, {
		method: method,
		headers: ajaxheader(false),
		body: body && JSON.stringify(body)
	});
	response.data = await response.json();
	return response;
};

// make authorized ajax-call with json data.
const fetchjsonauth = async (method, url, body) => {
	const resp0 = await fetch(url, { // 1-st try
		method: method,
		headers: ajaxheader(true),
		body: body && JSON.stringify(body)
	});
	if (resp0.status === 401 && auth.token.refrsh) { // Unauthorized
		const resp1 = await fetchjson("POST", "/api/auth/refrsh", {
			refrsh: auth.token.refrsh
		});
		const data = await resp1.json();
		if (!resp1.ok) {
			throw new HttpError(resp1.status, data);
		}
		auth.signin(data);
		const resp2 = fetch(url, { // 2-nd try
			method: method,
			headers: ajaxheader(true),
			body: body && JSON.stringify(body)
		});
		return resp2;
	}
	return resp0;
};

// make authorized ajax-call with json data and get json-response.
const fetchajaxauth = async (method, url, body) => {
	const response = await fetchjsonauth(method, url, body);
	response.data = await response.json();
	return response;
};

// show / hide global preloader.
let loadcount = 1; // ajax working request count
const viewpreloader = count => {
	loadcount += count;
	const prl = document.querySelector(".preloader-lock");
	if (loadcount > 0) {
		prl.style.display = "";
	} else {
		prl.style.display = "none";
	}
};

// The End.
