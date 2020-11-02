var $jscomp=$jscomp||{};$jscomp.scope={};$jscomp.arrayIteratorImpl=function(a){var b=0;return function(){return b<a.length?{done:!1,value:a[b++]}:{done:!0}}};$jscomp.arrayIterator=function(a){return{next:$jscomp.arrayIteratorImpl(a)}};$jscomp.makeIterator=function(a){var b="undefined"!=typeof Symbol&&Symbol.iterator&&a[Symbol.iterator];return b?b.call(a):$jscomp.arrayIterator(a)};$jscomp.arrayFromIterator=function(a){for(var b,c=[];!(b=a.next()).done;)c.push(b.value);return c};
$jscomp.arrayFromIterable=function(a){return a instanceof Array?a:$jscomp.arrayFromIterator($jscomp.makeIterator(a))};$jscomp.ASSUME_ES5=!1;$jscomp.ASSUME_NO_NATIVE_MAP=!1;$jscomp.ASSUME_NO_NATIVE_SET=!1;$jscomp.SIMPLE_FROUND_POLYFILL=!1;$jscomp.objectCreate=$jscomp.ASSUME_ES5||"function"==typeof Object.create?Object.create:function(a){var b=function(){};b.prototype=a;return new b};$jscomp.underscoreProtoCanBeSet=function(){var a={a:!0},b={};try{return b.__proto__=a,b.a}catch(c){}return!1};
$jscomp.setPrototypeOf="function"==typeof Object.setPrototypeOf?Object.setPrototypeOf:$jscomp.underscoreProtoCanBeSet()?function(a,b){a.__proto__=b;if(a.__proto__!==b)throw new TypeError(a+" is not extensible");return a}:null;
$jscomp.inherits=function(a,b){a.prototype=$jscomp.objectCreate(b.prototype);a.prototype.constructor=a;if($jscomp.setPrototypeOf){var c=$jscomp.setPrototypeOf;c(a,b)}else for(c in b)if("prototype"!=c)if(Object.defineProperties){var f=Object.getOwnPropertyDescriptor(b,c);f&&Object.defineProperty(a,c,f)}else a[c]=b[c];a.superClass_=b.prototype};$jscomp.getGlobal=function(a){return"undefined"!=typeof window&&window===a?a:"undefined"!=typeof global&&null!=global?global:a};$jscomp.global=$jscomp.getGlobal(this);
$jscomp.defineProperty=$jscomp.ASSUME_ES5||"function"==typeof Object.defineProperties?Object.defineProperty:function(a,b,c){a!=Array.prototype&&a!=Object.prototype&&(a[b]=c.value)};$jscomp.polyfill=function(a,b,c,f){if(b){c=$jscomp.global;a=a.split(".");for(f=0;f<a.length-1;f++){var d=a[f];d in c||(c[d]={});c=c[d]}a=a[a.length-1];f=c[a];b=b(f);b!=f&&null!=b&&$jscomp.defineProperty(c,a,{configurable:!0,writable:!0,value:b})}};$jscomp.FORCE_POLYFILL_PROMISE=!1;
$jscomp.polyfill("Promise",function(a){function b(){this.batch_=null}function c(a){return a instanceof d?a:new d(function(b,c){b(a)})}if(a&&!$jscomp.FORCE_POLYFILL_PROMISE)return a;b.prototype.asyncExecute=function(a){if(null==this.batch_){this.batch_=[];var b=this;this.asyncExecuteFunction(function(){b.executeBatch_()})}this.batch_.push(a)};var f=$jscomp.global.setTimeout;b.prototype.asyncExecuteFunction=function(a){f(a,0)};b.prototype.executeBatch_=function(){for(;this.batch_&&this.batch_.length;){var a=
this.batch_;this.batch_=[];for(var b=0;b<a.length;++b){var c=a[b];a[b]=null;try{c()}catch(l){this.asyncThrow_(l)}}}this.batch_=null};b.prototype.asyncThrow_=function(a){this.asyncExecuteFunction(function(){throw a;})};var d=function(a){this.state_=0;this.result_=void 0;this.onSettledCallbacks_=[];var b=this.createResolveAndReject_();try{a(b.resolve,b.reject)}catch(g){b.reject(g)}};d.prototype.createResolveAndReject_=function(){function a(a){return function(d){c||(c=!0,a.call(b,d))}}var b=this,c=!1;
return{resolve:a(this.resolveTo_),reject:a(this.reject_)}};d.prototype.resolveTo_=function(a){if(a===this)this.reject_(new TypeError("A Promise cannot resolve to itself"));else if(a instanceof d)this.settleSameAsPromise_(a);else{a:switch(typeof a){case "object":var b=null!=a;break a;case "function":b=!0;break a;default:b=!1}b?this.resolveToNonPromiseObj_(a):this.fulfill_(a)}};d.prototype.resolveToNonPromiseObj_=function(a){var b=void 0;try{b=a.then}catch(g){this.reject_(g);return}"function"==typeof b?
this.settleSameAsThenable_(b,a):this.fulfill_(a)};d.prototype.reject_=function(a){this.settle_(2,a)};d.prototype.fulfill_=function(a){this.settle_(1,a)};d.prototype.settle_=function(a,b){if(0!=this.state_)throw Error("Cannot settle("+a+", "+b+"): Promise already settled in state"+this.state_);this.state_=a;this.result_=b;this.executeOnSettledCallbacks_()};d.prototype.executeOnSettledCallbacks_=function(){if(null!=this.onSettledCallbacks_){for(var a=0;a<this.onSettledCallbacks_.length;++a)e.asyncExecute(this.onSettledCallbacks_[a]);
this.onSettledCallbacks_=null}};var e=new b;d.prototype.settleSameAsPromise_=function(a){var b=this.createResolveAndReject_();a.callWhenSettled_(b.resolve,b.reject)};d.prototype.settleSameAsThenable_=function(a,b){var c=this.createResolveAndReject_();try{a.call(b,c.resolve,c.reject)}catch(l){c.reject(l)}};d.prototype.then=function(a,b){function c(a,b){return"function"==typeof a?function(b){try{e(a(b))}catch(m){f(m)}}:b}var e,f,h=new d(function(a,b){e=a;f=b});this.callWhenSettled_(c(a,e),c(b,f));return h};
d.prototype.catch=function(a){return this.then(void 0,a)};d.prototype.callWhenSettled_=function(a,b){function c(){switch(d.state_){case 1:a(d.result_);break;case 2:b(d.result_);break;default:throw Error("Unexpected state: "+d.state_);}}var d=this;null==this.onSettledCallbacks_?e.asyncExecute(c):this.onSettledCallbacks_.push(c)};d.resolve=c;d.reject=function(a){return new d(function(b,c){c(a)})};d.race=function(a){return new d(function(b,d){for(var e=$jscomp.makeIterator(a),f=e.next();!f.done;f=e.next())c(f.value).callWhenSettled_(b,
d)})};d.all=function(a){var b=$jscomp.makeIterator(a),e=b.next();return e.done?c([]):new d(function(a,d){function f(b){return function(c){h[b]=c;g--;0==g&&a(h)}}var h=[],g=0;do h.push(void 0),g++,c(e.value).callWhenSettled_(f(h.length-1),d),e=b.next();while(!e.done)})};return d},"es6","es3");$jscomp.SYMBOL_PREFIX="jscomp_symbol_";$jscomp.initSymbol=function(){$jscomp.initSymbol=function(){};$jscomp.global.Symbol||($jscomp.global.Symbol=$jscomp.Symbol)};
$jscomp.SymbolClass=function(a,b){this.$jscomp$symbol$id_=a;$jscomp.defineProperty(this,"description",{configurable:!0,writable:!0,value:b})};$jscomp.SymbolClass.prototype.toString=function(){return this.$jscomp$symbol$id_};$jscomp.Symbol=function(){function a(c){if(this instanceof a)throw new TypeError("Symbol is not a constructor");return new $jscomp.SymbolClass($jscomp.SYMBOL_PREFIX+(c||"")+"_"+b++,c)}var b=0;return a}();
$jscomp.initSymbolIterator=function(){$jscomp.initSymbol();var a=$jscomp.global.Symbol.iterator;a||(a=$jscomp.global.Symbol.iterator=$jscomp.global.Symbol("Symbol.iterator"));"function"!=typeof Array.prototype[a]&&$jscomp.defineProperty(Array.prototype,a,{configurable:!0,writable:!0,value:function(){return $jscomp.iteratorPrototype($jscomp.arrayIteratorImpl(this))}});$jscomp.initSymbolIterator=function(){}};
$jscomp.initSymbolAsyncIterator=function(){$jscomp.initSymbol();var a=$jscomp.global.Symbol.asyncIterator;a||(a=$jscomp.global.Symbol.asyncIterator=$jscomp.global.Symbol("Symbol.asyncIterator"));$jscomp.initSymbolAsyncIterator=function(){}};$jscomp.iteratorPrototype=function(a){$jscomp.initSymbolIterator();a={next:a};a[$jscomp.global.Symbol.iterator]=function(){return this};return a};$jscomp.generator={};
$jscomp.generator.ensureIteratorResultIsObject_=function(a){if(!(a instanceof Object))throw new TypeError("Iterator result "+a+" is not an object");};$jscomp.generator.Context=function(){this.isRunning_=!1;this.yieldAllIterator_=null;this.yieldResult=void 0;this.nextAddress=1;this.finallyAddress_=this.catchAddress_=0;this.finallyContexts_=this.abruptCompletion_=null};
$jscomp.generator.Context.prototype.start_=function(){if(this.isRunning_)throw new TypeError("Generator is already running");this.isRunning_=!0};$jscomp.generator.Context.prototype.stop_=function(){this.isRunning_=!1};$jscomp.generator.Context.prototype.jumpToErrorHandler_=function(){this.nextAddress=this.catchAddress_||this.finallyAddress_};$jscomp.generator.Context.prototype.next_=function(a){this.yieldResult=a};
$jscomp.generator.Context.prototype.throw_=function(a){this.abruptCompletion_={exception:a,isException:!0};this.jumpToErrorHandler_()};$jscomp.generator.Context.prototype.return=function(a){this.abruptCompletion_={return:a};this.nextAddress=this.finallyAddress_};$jscomp.generator.Context.prototype.jumpThroughFinallyBlocks=function(a){this.abruptCompletion_={jumpTo:a};this.nextAddress=this.finallyAddress_};$jscomp.generator.Context.prototype.yield=function(a,b){this.nextAddress=b;return{value:a}};
$jscomp.generator.Context.prototype.yieldAll=function(a,b){a=$jscomp.makeIterator(a);var c=a.next();$jscomp.generator.ensureIteratorResultIsObject_(c);if(c.done)this.yieldResult=c.value,this.nextAddress=b;else return this.yieldAllIterator_=a,this.yield(c.value,b)};$jscomp.generator.Context.prototype.jumpTo=function(a){this.nextAddress=a};$jscomp.generator.Context.prototype.jumpToEnd=function(){this.nextAddress=0};
$jscomp.generator.Context.prototype.setCatchFinallyBlocks=function(a,b){this.catchAddress_=a;void 0!=b&&(this.finallyAddress_=b)};$jscomp.generator.Context.prototype.setFinallyBlock=function(a){this.catchAddress_=0;this.finallyAddress_=a||0};$jscomp.generator.Context.prototype.leaveTryBlock=function(a,b){this.nextAddress=a;this.catchAddress_=b||0};
$jscomp.generator.Context.prototype.enterCatchBlock=function(a){this.catchAddress_=a||0;a=this.abruptCompletion_.exception;this.abruptCompletion_=null;return a};$jscomp.generator.Context.prototype.enterFinallyBlock=function(a,b,c){c?this.finallyContexts_[c]=this.abruptCompletion_:this.finallyContexts_=[this.abruptCompletion_];this.catchAddress_=a||0;this.finallyAddress_=b||0};
$jscomp.generator.Context.prototype.leaveFinallyBlock=function(a,b){b=this.finallyContexts_.splice(b||0)[0];if(b=this.abruptCompletion_=this.abruptCompletion_||b){if(b.isException)return this.jumpToErrorHandler_();void 0!=b.jumpTo&&this.finallyAddress_<b.jumpTo?(this.nextAddress=b.jumpTo,this.abruptCompletion_=null):this.nextAddress=this.finallyAddress_}else this.nextAddress=a};$jscomp.generator.Context.prototype.forIn=function(a){return new $jscomp.generator.Context.PropertyIterator(a)};
$jscomp.generator.Context.PropertyIterator=function(a){this.object_=a;this.properties_=[];for(var b in a)this.properties_.push(b);this.properties_.reverse()};$jscomp.generator.Context.PropertyIterator.prototype.getNext=function(){for(;0<this.properties_.length;){var a=this.properties_.pop();if(a in this.object_)return a}return null};$jscomp.generator.Engine_=function(a){this.context_=new $jscomp.generator.Context;this.program_=a};
$jscomp.generator.Engine_.prototype.next_=function(a){this.context_.start_();if(this.context_.yieldAllIterator_)return this.yieldAllStep_(this.context_.yieldAllIterator_.next,a,this.context_.next_);this.context_.next_(a);return this.nextStep_()};
$jscomp.generator.Engine_.prototype.return_=function(a){this.context_.start_();var b=this.context_.yieldAllIterator_;if(b)return this.yieldAllStep_("return"in b?b["return"]:function(a){return{value:a,done:!0}},a,this.context_.return);this.context_.return(a);return this.nextStep_()};
$jscomp.generator.Engine_.prototype.throw_=function(a){this.context_.start_();if(this.context_.yieldAllIterator_)return this.yieldAllStep_(this.context_.yieldAllIterator_["throw"],a,this.context_.next_);this.context_.throw_(a);return this.nextStep_()};
$jscomp.generator.Engine_.prototype.yieldAllStep_=function(a,b,c){try{var f=a.call(this.context_.yieldAllIterator_,b);$jscomp.generator.ensureIteratorResultIsObject_(f);if(!f.done)return this.context_.stop_(),f;var d=f.value}catch(e){return this.context_.yieldAllIterator_=null,this.context_.throw_(e),this.nextStep_()}this.context_.yieldAllIterator_=null;c.call(this.context_,d);return this.nextStep_()};
$jscomp.generator.Engine_.prototype.nextStep_=function(){for(;this.context_.nextAddress;)try{var a=this.program_(this.context_);if(a)return this.context_.stop_(),{value:a.value,done:!1}}catch(b){this.context_.yieldResult=void 0,this.context_.throw_(b)}this.context_.stop_();if(this.context_.abruptCompletion_){a=this.context_.abruptCompletion_;this.context_.abruptCompletion_=null;if(a.isException)throw a.exception;return{value:a.return,done:!0}}return{value:void 0,done:!0}};
$jscomp.generator.Generator_=function(a){this.next=function(b){return a.next_(b)};this.throw=function(b){return a.throw_(b)};this.return=function(b){return a.return_(b)};$jscomp.initSymbolIterator();this[Symbol.iterator]=function(){return this}};$jscomp.generator.createGenerator=function(a,b){b=new $jscomp.generator.Generator_(new $jscomp.generator.Engine_(b));$jscomp.setPrototypeOf&&$jscomp.setPrototypeOf(b,a.prototype);return b};
$jscomp.asyncExecutePromiseGenerator=function(a){function b(b){return a.next(b)}function c(b){return a.throw(b)}return new Promise(function(f,d){function e(a){a.done?f(a.value):Promise.resolve(a.value).then(b,c).then(e,d)}e(a.next())})};$jscomp.asyncExecutePromiseGeneratorFunction=function(a){return $jscomp.asyncExecutePromiseGenerator(a())};$jscomp.asyncExecutePromiseGeneratorProgram=function(a){return $jscomp.asyncExecutePromiseGenerator(new $jscomp.generator.Generator_(new $jscomp.generator.Engine_(a)))};
$jscomp.checkStringArgs=function(a,b,c){if(null==a)throw new TypeError("The 'this' value for String.prototype."+c+" must not be null or undefined");if(b instanceof RegExp)throw new TypeError("First argument to String.prototype."+c+" must not be a regular expression");return a+""};
$jscomp.polyfill("String.prototype.repeat",function(a){return a?a:function(a){var b=$jscomp.checkStringArgs(this,null,"repeat");if(0>a||1342177279<a)throw new RangeError("Invalid count value");a|=0;for(var f="";a;)if(a&1&&(f+=b),a>>>=1)b+=b;return f}},"es6","es3");$jscomp.polyfill("String.prototype.trimRight",function(a){function b(){return this.replace(/[\s\xa0]+$/,"")}return a||b},"es_2019","es3");var buildvers="0.5.6",builddate="2020.11.02",devmode=!1,traceajax=function(){};if("undefined"===typeof $.fn.popover)throw Error("Bootstrap library required");String.prototype.format||(String.prototype.format=function(){var a=arguments;return this.replace(/{(\d+)}/g,function(b,c){return"undefined"!==typeof a[c]?a[c]:b})});
String.prototype.printf||(String.prototype.printf=function(){var a=Array.prototype.slice.call(arguments),b=-1;return this.replace(/%(-)?(0?[0-9]+)?([.][0-9]+)?([#][0-9]+)?([scfpexd%])/g,function(c,f,d,e,h,k){if("%%"===c)return"%";if(void 0!==a[++b]){c=e?parseInt(e.substr(1)):void 0;e=h?parseInt(h.substr(1)):void 0;switch(k){case "s":var g=a[b];break;case "c":g=a[b][0];break;case "f":g=parseFloat(a[b]).toFixed(c);break;case "p":g=parseFloat(a[b]).toPrecision(c);break;case "e":g=parseFloat(a[b]).toExponential(c);
break;case "x":g=parseInt(a[b]).toString(e?e:16);break;case "d":g=parseFloat(parseInt(a[b],e?e:10).toPrecision(c)).toFixed(0)}g="object"===typeof g?JSON.stringify(g):g.toString(e);c=parseInt(d);for(d=d&&"0"===d[0]?"0":" ";g.length<c;)g=void 0!==f?g+d:d+g;return g}})});
var fmtfilesize=function(a){return 1536>a?a+" bytes":1048576>a?(a/1024).toPrecision(3)+" kB":1073741824>a?(a/1048576).toPrecision(3)+" MB":1099511627776>a?(a/1073741824).toPrecision(3)+" GB":(a/1099511627776).toPrecision(3)+" TB"},fmtitemsize=function(a){return 1536>a?fmtfilesize(a):"%s (%d bytes)".printf(fmtfilesize(a),a)},fmttime=function(a,b){var c=function(a,b){a=Math.floor(a).toString();b-=a.length;return 0<b?"0".repeat(b)+a:a};if(60>b)return c(a,2);if(3600>b)return b=a%60,c(a/60,2)+":"+c(b,
2);b=a%60;var f=a%3600/60;return c(a/3600,2)+":"+c(f,2)+":"+c(b,2)},makestrid=function(a){for(var b="",c=0;c<a;c++)b+="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789".charAt(Math.floor(62*Math.random()));return b},makeeventmodel=function(){var a=[];return{emit:function(b,c){for(var f=[],d=1;d<arguments.length;++d)f[d-1]=arguments[d];for(d=0;d<a.length;){var e=$jscomp.makeIterator(a[d]),h=e.next().value,k=e.next().value;e=e.next().value;if(h===b){e&&(a.splice(d,1),d--);try{k.apply(null,
$jscomp.arrayFromIterable(f))}catch(g){console.error(g)}}d++}},on:function(b,c,f){return a.push([b,c,void 0===f?!1:f])},once:function(b,c){return a.push([b,c,!0])},off:function(b,c){for(var f=0;f<a.length;){var d=$jscomp.makeIterator(a[f]),e=d.next().value;d=d.next().value;e!==b&&b||d!==c&&c?f++:a.splice(f,1)}},onmap:function(b){for(var c in b)a.push([c,b[c],!1])},offmap:function(b){for(var c in b){var f=b[c],d;for(d in a.length){var e=$jscomp.makeIterator(a[d]),h=e.next().value;e=e.next().value;
if(h===c&&e===f){a.splice(d,1);break}}}},listens:function(b,c){for(var f=0,d=$jscomp.makeIterator(a),e=d.next();!e.done;e=d.next()){var h=$jscomp.makeIterator(e.value);e=h.next().value;h=h.next().value;e!==b&&b||h!==c&&c||f++}return f},listenlen:function(){return a.length}}},extend=function(a,b){for(var c in b)a[c]=b[c];return a};var auth=extend({token:{access:null,refrsh:null},login:"",signed:function(){return!!this.token.access},claims:function(){try{var a=this.token.access.split(".");return JSON.parse(atob(a[1]))}catch(b){return null}},signin:function(a,b){sessionStorage.setItem("token",JSON.stringify(a));this.token.access=a.access;this.token.refrsh=a.refrsh;b&&(sessionStorage.setItem("login",b),this.login=b);this.emit("auth",!0)},signout:function(){sessionStorage.removeItem("token");this.token.access=null;this.token.refrsh=
null;this.emit("auth",!1)},signload:function(){try{var a=JSON.parse(sessionStorage.getItem("token"));this.token.access=a.access;this.token.refrsh=a.refrsh;this.login=sessionStorage.getItem("login")||"";this.emit("auth",!0)}catch(b){this.token.access=null,this.token.refrsh=null,this.login="",this.emit("auth",!1)}}},makeeventmodel()),ajaxcc=extend({},makeeventmodel()),HttpError=function(a,b){var c=Error.call(this,b.what);this.message=c.message;"stack"in c&&(this.stack=c.stack);this.name="HttpError";
this.status=a;extend(this,b)};$jscomp.inherits(HttpError,Error);
var ajaxheader=function(a){var b={Accept:"application/json;charset=utf-8","Content-Type":"application/json;charset=utf-8"};a&&auth.token.access&&(b.Authorization="Bearer "+auth.token.access);return b},fetchjson=function(a,b,c){return $jscomp.asyncExecutePromiseGeneratorProgram(function(f){return 1==f.nextAddress?f.yield(fetch(b,{method:a,headers:ajaxheader(!1),body:JSON.stringify(c)}),2):f.return(f.yieldResult)})},fetchajax=function(a,b,c){var f,d;return $jscomp.asyncExecutePromiseGeneratorProgram(function(e){if(1==
e.nextAddress)return e.yield(fetch(b,{method:a,headers:ajaxheader(!1),body:c&&JSON.stringify(c)}),2);if(3!=e.nextAddress)return d=f=e.yieldResult,e.yield(f.json(),3);d.data=e.yieldResult;return e.return(f)})},fetchjsonauth=function(a,b,c){var f,d,e,h;return $jscomp.asyncExecutePromiseGeneratorProgram(function(k){switch(k.nextAddress){case 1:return k.yield(fetch(b,{method:a,headers:ajaxheader(!0),body:c&&JSON.stringify(c)}),2);case 2:f=k.yieldResult;if(401!==f.status||!auth.token.refrsh){k.jumpTo(3);
break}return k.yield(fetchjson("POST","/api/auth/refrsh",{refrsh:auth.token.refrsh}),4);case 4:return d=k.yieldResult,k.yield(d.json(),5);case 5:e=k.yieldResult;if(!d.ok)throw new HttpError(d.status,e);auth.signin(e);h=fetch(b,{method:a,headers:ajaxheader(!0),body:c&&JSON.stringify(c)});return k.return(h);case 3:return k.return(f)}})},fetchajaxauth=function(a,b,c){var f,d;return $jscomp.asyncExecutePromiseGeneratorProgram(function(e){if(1==e.nextAddress)return e.yield(fetchjsonauth(a,b,c),2);if(3!=
e.nextAddress)return d=f=e.yieldResult,e.yield(f.json(),3);d.data=e.yieldResult;return e.return(f)})};var scanfreq=2E3,app=new Vue({el:"#app",template:"#app-tpl",data:{srvinf:{},memgc:{},cchinf:{},log:[],timemode:1},computed:{consolecontent:function(){for(var a=[],b=$jscomp.makeIterator(this.log),c=b.next();!c.done;c=b.next()){c=c.value;var f="",d=new Date(c.time);switch(this.timemode){case 1:f=d.toLocaleTimeString()+" ";break;case 2:f=d.toLocaleString()+" "}c.file&&(f+=c.file+":"+c.line+": ");a.unshift(f+c.msg.trimRight())}return a.join("\n")},isnoprefix:function(){return 0===this.timemode&&"btn-info"||
"btn-outline-info"},istime:function(){return 1===this.timemode&&"btn-info"||"btn-outline-info"},isdatetime:function(){return 2===this.timemode&&"btn-info"||"btn-outline-info"},avrshow:function(){return 1<(this.cchinf.tmbjpgnum?1:0)+(this.cchinf.tmbpngnum?1:0)+(this.cchinf.tmbgifnum?1:0)},avrtmbcchsize:function(){return this.cchinf.tmbcchnum?(this.cchinf.tmbcchsize1/this.cchinf.tmbcchnum).toFixed():"N/A"},avrtmbjpgsize:function(){return this.cchinf.tmbjpgnum?(this.cchinf.tmbjpgsize1/this.cchinf.tmbjpgnum).toFixed():
"N/A"},avrtmbpngsize:function(){return this.cchinf.tmbpngnum?(this.cchinf.tmbpngsize1/this.cchinf.tmbpngnum).toFixed():"N/A"},avrtmbgifsize:function(){return this.cchinf.tmbgifnum?(this.cchinf.tmbgifsize1/this.cchinf.tmbgifnum).toFixed():"N/A"}},methods:{fmtduration:function(a){return 864E5<a?"%d days %02d hours %02d min".printf(Math.floor(a/864E5),Math.floor(a%864E5/36E5),Math.floor(a%36E5/6E4)):36E5<a?"%d hours %02d min %02d sec".printf(Math.floor(a/36E5),Math.floor(a%36E5/6E4),Math.floor(a%6E4/
1E3)):"%02d min %02d sec".printf(Math.floor(a%36E5/6E4),Math.floor(a%6E4/1E3))},ongetlog:function(){var a=this;(function(){var b,c,f;return $jscomp.asyncExecutePromiseGeneratorProgram(function(d){switch(d.nextAddress){case 1:return d.setCatchFinallyBlocks(2),d.yield(fetch("/api/stat/getlog"),4);case 4:b=d.yieldResult;if(!b.ok){d.jumpTo(5);break}c=a;return d.yield(b.json(),6);case 6:c.log=d.yieldResult;case 5:d.leaveTryBlock(0);break;case 2:f=d.enterCatchBlock(),console.error(f),d.jumpToEnd()}})})()},
onnoprefix:function(){this.timemode=0},ontime:function(){this.timemode=1},ondatetime:function(){this.timemode=2}},mounted:function(){var a=this;(function(){var b,c,f;return $jscomp.asyncExecutePromiseGeneratorProgram(function(d){switch(d.nextAddress){case 1:return d.setCatchFinallyBlocks(2),d.yield(fetch("/api/stat/srvinf"),4);case 4:b=d.yieldResult;if(!b.ok){d.jumpTo(5);break}c=a;return d.yield(b.json(),6);case 6:c.srvinf=d.yieldResult,a.srvinf.buildvers=buildvers,a.srvinf.builddate=builddate;case 5:d.leaveTryBlock(0);
break;case 2:f=d.enterCatchBlock(),console.error(f),d.jumpToEnd()}})})();$("#collapse-memory").on("show.bs.collapse",function(){var b=!0;(function(){var c,f,d;return $jscomp.asyncExecutePromiseGeneratorProgram(function(e){switch(e.nextAddress){case 1:e.setCatchFinallyBlocks(2);case 4:if(!b){e.leaveTryBlock(0);break}return e.yield(fetch("/api/stat/memusg"),6);case 6:c=e.yieldResult;if(!c.ok){e.jumpTo(7);break}f=a;return e.yield(c.json(),8);case 8:f.memgc=e.yieldResult;case 7:return e.yield(new Promise(function(a){return setTimeout(a,
scanfreq)}),4);case 2:d=e.enterCatchBlock(),console.error(d),e.jumpToEnd()}})})();$("#collapse-memory").one("hide.bs.collapse",function(){b=!1})});$("#collapse-cache").on("show.bs.collapse",function(){var b=!0;(function(){var c,f,d;return $jscomp.asyncExecutePromiseGeneratorProgram(function(e){switch(e.nextAddress){case 1:e.setCatchFinallyBlocks(2);case 4:if(!b){e.leaveTryBlock(0);break}return e.yield(fetch("/api/stat/cchinf"),6);case 6:c=e.yieldResult;if(!c.ok){e.jumpTo(7);break}f=a;return e.yield(c.json(),
8);case 8:f.cchinf=e.yieldResult;case 7:return e.yield(new Promise(function(a){return setTimeout(a,scanfreq)}),4);case 2:d=e.enterCatchBlock(),console.error(d),e.jumpToEnd()}})})();$("#collapse-memory").one("hide.bs.collapse",function(){b=!1})});$("#collapse-console").on("show.bs.collapse",function(){a.ongetlog()})},beforeDestroy:function(){$("#collapse-memory").off("show.bs.collapse");$("#collapse-memory").off("hide.bs.collapse");$("#collapse-cache").off("show.bs.collapse");$("#collapse-cache").off("hide.bs.collapse");
$("#collapse-console").off("show.bs.collapse")}});$(document).ready(function(){$(".preloader-lock").hide("fast");$("#app").show("fast")});
