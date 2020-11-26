"use strict";

const photofilter = file => FTtoFV[file.type] === FV.image;

Vue.component('thumbslider-tag', {
	template: '#thumbslider-tpl',
	props: ["selfile", "list"],
	computed: {
		slide() {
			const lst = [];
			for (const file of this.list) {
				if (photofilter(file)) {
					lst.push(file);
				}
			}
			return lst;
		}
	},
	methods: {
		getthumb(file) {
			return filetmbpng(file, iconmapping);
		},

		onprev() {
			this.$refs.slide.scrollBy({ left: -120, behavior: 'smooth' });
		},
		onnext() {
			this.$refs.slide.scrollBy({ left: +120, behavior: 'smooth' });
		},
		onthumb(file) {
			this.$emit('select', file);
		}
	}
});

Vue.component('photoslider-tag', {
	template: '#photoslider-tpl',
	props: ["list"],
	data: function () {
		return {
			selfile: null,
			selfileurl: null,
			imgloading: false,
			repeatmode: 0 // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
		};
	},
	computed: {
		// index of selected file
		selfilepos() {
			for (const i in this.list) {
				if (this.selfile.puid === this.list[i].puid) {
					return Number(i);
				}
			}
		},
		// returns previous file in list
		getprev() {
			const prevpos = (from, to) => {
				for (let i = from - 1; i > to; i--) {
					const file = this.list[i];
					if (photofilter(file)) {
						return file;
					}
				}
			};
			return this.selfile && (prevpos(this.selfilepos, -1) || this.repeatmode === 2 && prevpos(this.list.length, this.selfilepos));
		},
		// returns next file in list
		getnext() {
			const nextpos = (from, to) => {
				for (let i = from + 1; i < to; i++) {
					const file = this.list[i];
					if (photofilter(file)) {
						return file;
					}
				}
			};
			return this.selfile && (nextpos(this.selfilepos, this.list.length) || this.repeatmode === 2 && nextpos(-1, this.selfilepos));
		},
		islist() {
			return this.list && this.list.length > 1;
		}
	},
	methods: {
		load(url) {
			if (url) {
				this.imgloading = true;
			}
			this.selfileurl = url;
		},
		select(file) {
			this.selfile = file;
			this.load(mediaurl(file));
			this.$emit('select', file);
		},
		popup(file) {
			this.selfile = file;
			this.load(mediaurl(file));
			$(this.$refs.modal).modal('show');
		},
		close() {
			this.selfile = null;
			this.load(mediaurl(file));
			$(this.$refs.modal).modal('hide');
		},

		onimgload(e) {
			this.imgloading = false;
		},
		onimgerror(e) {
			this.imgloading = false;
		},
		onkeyup(e) {
			switch (e.code) {
				case 'ArrowLeft':
					this.onprev();
					break;
				case 'ArrowRight':
				case 'Space':
					this.onnext();
					break;
				case 'Home':
					if (this.list) {
						this.select(this.list[0]);
					}
					break;
				case 'End':
					if (this.list) {
						this.select(this.list[this.list.length - 1]);
					}
					break;
			}
		},
		onselect(file) {
			this.select(file);
		},
		onprev() {
			if (this.getprev) {
				this.select(this.getprev);
			}
		},
		onnext() {
			if (this.getnext) {
				this.select(this.getnext);
			}
		},
		onclose() {
			this.close();
		}
	}
});

// The End.
