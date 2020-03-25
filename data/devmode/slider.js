"use strict";

const photofilter = file => FTtoFV[file.type] === FV.image || FTtoFV[file.type] === FV.video;

Vue.component('slider-tag', {
	template: '#slider-tpl',
	props: ["list"],
	data: function () {
		return {
			visible: false,
			selfile: null,
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

		curimg() {
			return this.selfile && getfileurl(this.selfile);
		}
	},
	methods: {
		popup(file) {
			this.selfile = file;
			this.visible = true;
			Vue.nextTick(() => {
				this.$refs.wall.focus();
			});
		},

		onkeyup(e) {
			switch (e.code) {
				case 'Escape':
					this.visible = false;
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
						this.selfile = this.list[0];
						this.$emit('select', this.selfile);
					}
					break;
				case 'End':
					if (this.list) {
						this.selfile = this.list[this.list.length - 1];
						this.$emit('select', this.selfile);
					}
					break;
			}
		},
		onprev() {
			if (this.getprev) {
				this.selfile = this.getprev;
				this.$emit('select', this.selfile);
			}
		},
		onnext() {
			if (this.getnext) {
				this.selfile = this.getnext;
				this.$emit('select', this.selfile);
			}
		},
		onclose() {
			this.visible = false;
		}
	}
});

// The End.
