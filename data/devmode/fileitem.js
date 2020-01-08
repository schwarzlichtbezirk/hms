"use strict";

const G = {
};

const globalmixin = {
	data: function () {
		return G;
	},
	computed: {
	},
	methods: {
	}
};

Vue.component('file-icon-tag', {
	template: `
<div class="file-item" v-bind:class="itemview" v-on:click.stop="onselect" v-on:dblclick.stop="onrun" v-bind:title="fmttitle">
	<div class="img-overlay">
		<picture>
			<source v-bind:srcset="webpicon" type="image/webp">
			<source v-bind:srcset="pngicon" type="image/png">
			<img src="/asst/file-png/doc-file.png" v-bind:alt="file.name">
		</picture>
		<picture v-if="file.pref">
			<source srcset="/asst/file-webp/shared.webp" type="image/webp">
			<source srcset="/asst/file-png/shared.png" type="image/png">
			<img src="/asst/file-png/shared.png" alt="shared">
		</picture>
		<img v-if="state.playback" src="/asst/equalizer-bars.gif" alt="shared">
	</div>
	<p>{{file.name}}</p>
</div>
`,
	props: ["file", "state"],
	computed: {
		fmttitle() {
			let title = this.file.name;
			if (this.file.pref) {
				title += '\nshare: ' + shareprefix + this.file.pref;
			}
			if (this.file.type !== FT.dir) {
				title += '\nsize: ' + fmtitemsize(this.file.size);
			}
			return title;
		},

		webpicon() {
			return '/asst/file-webp/' + geticonname(this.file) + '.webp';
		},

		pngicon() {
			return '/asst/file-png/' + geticonname(this.file) + '.png';
		},

		// manage items classes
		itemview() {
			return { selected: this.state.selected };
		}
	},
	methods: {
		onselect() {
			this.$emit('select', this.file);
		},

		onrun() {
			this.$emit('run', this.file);
		}
	}
});

// <file-page-tag v-bind:list="playlist">files</file-page-tag>
Vue.component('file-page-tag', {
	template: `
<div class="card">
	<div class="card-header">
		<a class="card-link d-flex w-100" data-toggle="collapse" v-bind:href="'#body'+iid"><slot>files</slot></a>
	</div>
	<div v-bind:id="'body'+iid" class="collapse show">
		<div class="card-body folder-list" v-on:click="ondelsel">
			<template v-for="file in list">
				<file-icon-tag v-bind:file="file" v-bind:state="{selected:selected===file,playback:playbackfile===file}" v-on:select="onfilesel" v-on:run="onfilerun" />
			</template>
		</div>
	</div>
</div>
`,
	props: ["list"],
	data: function () {
		return {
			iid: makestrid(10) // instance ID
		};
	},
	computed: {
	},
	methods: {
	}
});

// The End.
