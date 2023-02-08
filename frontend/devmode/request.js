"use strict";
// All what is need for ajax.

const eventHub = makeeventhub();

const makeauth = () => {
	let access = null;
	let refrsh = null;
	let login = "";

	const t = {
		get signed() {
			return !!access;
		},
		get access() {
			return access;
		},
		get refrsh() {
			return refrsh;
		},
		get login() {
			return login;
		},
		get uid() {
			return this.claims()?.uid;
		},
		claims() {
			try {
				const p = access.split('.');
				return JSON.parse(atob(p[1]));
			} catch {
				return null;
			}
		},
		signin(ta, tr, ln) {
			sessionStorage.setItem('token.access', ta);
			sessionStorage.setItem('token.refrsh', tr);
			access = ta;
			refrsh = tr;
			if (ln) {
				sessionStorage.setItem('login', ln);
				login = ln;
			}
			eventHub.emit('auth', this);
		},
		signout() {
			sessionStorage.removeItem('token.access');
			sessionStorage.removeItem('token.refrsh');
			access = null;
			refrsh = null;
			// login remains unchanged
			eventHub.emit('auth', this);
		},
		signload() {
			access = storageGetString('token.access', null);
			refrsh = storageGetString('token.refrsh', null);
			login = storageGetString('login', "");
			eventHub.emit('auth', this);
		}
	};

	return t;
};

const auth = makeauth();

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
		'Accept': 'application/json',
		'Content-Type': 'application/json; charset=utf-8',
	};
	if (bearer && auth.access) {
		hdr['Authorization'] = 'Bearer ' + auth.access;
	}
	return hdr;
};

// make ajax-call with json data.
const fetchjson = async (method, url, body) => {
	return await fetch(url, {
		method: method,
		headers: ajaxheader(false),
		body: body && JSON.stringify(body),
	});
};

// make authorized ajax-call with json data.
const fetchjsonauth = async (method, url, body) => {
	const resp0 = await fetch(url, { // 1-st try
		method: method,
		headers: ajaxheader(true),
		body: body && JSON.stringify(body),
	});
	if (resp0.status === 401 && auth.refrsh) { // Unauthorized
		const resp1 = await fetch("/api/auth/refrsh", { // get new token
			method: "POST",
			headers: ajaxheader(true),
			body: JSON.stringify({
				refrsh: auth.refrsh
			}),
		});
		const data1 = await resp1.json();
		if (!resp1.ok) {
			throw new HttpError(resp1.status, data1);
		}
		auth.signin(data1.access, data1.refrsh);
		const resp2 = fetch(url, { // 2-nd try
			method: method,
			headers: ajaxheader(true),
			body: body && JSON.stringify(body),
		});
		return resp2;
	}
	return resp0;
};

// show / hide global preloader.
let loadcount = 1; // ajax working request count
const viewpreloader = count => {
	loadcount += count;
	const prl = document.querySelector(".preloader-lock");
	if (prl) {
		if (loadcount > 0) {
			prl.style.display = '';
		} else {
			prl.style.display = 'none';
		}
	}
};

// The End.
