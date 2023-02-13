"use strict";

// This file is included only for developer mode linkage

const buildvers = "0.9.3";
const builddate = "2023.02.13";
console.info("version: %s, builton: %s", buildvers, builddate);
console.info("starts in developer mode");

const devmode = true;

const traceajax = (response, data) => {
	if (response.ok) {
		console.log(response.status, response.url);
	}
	if (data) {
		console.log(data);
	}
};

// The End.
