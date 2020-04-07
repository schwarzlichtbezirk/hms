"use strict";

// This file is included only for developer mode linkage

const buildvers = "0.4.1";
const builddate = "2020.04.07";
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

// The End.
