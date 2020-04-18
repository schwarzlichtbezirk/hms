"use strict";

// This file is included only for developer mode linkage

const buildvers = "0.4.2";
const builddate = "2020.04.16";
console.info("version: %s, builton: %s", buildvers, builddate);
console.info("starts in developer mode");

const devmode = true;

const traceresponse = xhr => {
	if (xhr.status >= 200 && xhr.status < 300) {
		console.log(xhr.status, xhr.responseURL);
	}
	if (xhr.response) {
		console.log(xhr.response);
	}
};

const traceajax = response => {
	if (response.status >= 200 && response.status < 300) {
		console.log(response.status, response.url);
	}
	if (response.data) {
		console.log(response.data);
	}
};

// The End.
