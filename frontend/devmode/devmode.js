"use strict";

// This file is included only for developer mode linkage

const buildvers = "0.7.5";
const builddate = "2021.07.10";
console.info("version: %s, builton: %s", buildvers, builddate);
console.info("starts in developer mode");

const devmode = true;

const traceajax = (response, data) => {
	if (response.ok) {
		console.log(response.status, response.url);
	}
	if (response.data || data) {
		console.log(response.data || data);
	}
};

// The End.
