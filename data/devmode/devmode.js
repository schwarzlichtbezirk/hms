"use strict";

// This file is included only for developer mode linkage

const buildvers = "0.4.3";
const builddate = "2020.06.03";
console.info("version: %s, builton: %s", buildvers, builddate);
console.info("starts in developer mode");

const devmode = true;

const traceajax = response => {
	if (response.ok) {
		console.log(response.status, response.url);
	}
	if (response.data) {
		console.log(response.data);
	}
};

// The End.
