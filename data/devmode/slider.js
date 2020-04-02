"use strict";

const photofilter = file => FTtoFV[file.type] === FV.image || FTtoFV[file.type] === FV.video;

Vue.component('thumbslider-tag', {
	template: '#thumbslider-tpl',
	props: ["selfile", "list"],
	computed: {
		slide() {
			const lst = [];
			for (const file of this.list) {
				if (file.ntmb === 1) {
					lst.push(file);
				}
			}
			return lst;
		}
	},
	methods: {
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
			visible: false,
			selfile: null,
			selfileurl: null,
			repeatmode: 0 // 0 - no any repeat, 1 - repeat single, 2 - repeat playlist
		};
	},
	computed: {
		// index of selected file
		selfilepos() {
			for (const i in this.list) {
				if (this.selfile.path === this.list[i].path) {
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
		select(file) {
			this.selfile = file;
			this.selfileurl = getfileurl(file);
			this.$emit('select', file);
		},
		popup(file) {
			this.selfile = file;
			this.selfileurl = getfileurl(file);
			this.visible = true;
			Vue.nextTick(() => {
				this.$refs.wall.focus();
			});
		},
		close() {
			this.selfile = null;
			this.selfileurl = null;
			this.visible = false;
		},

		onkeyup(e) {
			switch (e.code) {
				case 'Escape':
					this.close();
					break;
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
