"use strict";

// This file is included only for developer mode linkage

const buildvers = "0.1.5";
const builddate = "2020.01.10";
console.info("version: %s, builton: %s", buildvers, builddate);
console.info("starts in developer mode");

const devmode = true;

const traceresponse = (xhr) => {
	console.log(xhr.status, xhr.responseURL);
	if (xhr.response) {
		if (xhr.status === 200) { // OK
			console.log(xhr.response);
		} else {
			console.log(JSON.stringify(xhr.response).substring(0, 256));
		}
	}
};

// The End.
