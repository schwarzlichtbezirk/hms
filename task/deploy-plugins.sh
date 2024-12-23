#!/bin/bash -u
# This script downloads all javascript dependencies used on frontend.

plugdir=$(realpath -s "$(dirname $0)/../frontend/plugin")
mkdir -pv "$plugdir/images"

# bootstrap 5.3.2
# https://cdnjs.com/libraries/bootstrap
vers="5.3.2"
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/$vers/js/bootstrap.min.js --output $plugdir/bootstrap.min.js
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/$vers/js/bootstrap.min.js.map --output $plugdir/bootstrap.min.js.map
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/$vers/css/bootstrap.min.css --output $plugdir/bootstrap.min.css
curl https://cdnjs.cloudflare.com/ajax/libs/bootstrap/$vers/css/bootstrap.min.css.map --output $plugdir/bootstrap.min.css.map

# popper 2.11.8
# https://cdnjs.com/libraries/popper.js
vers="2.11.8"
curl https://cdnjs.cloudflare.com/ajax/libs/popper.js/$vers/umd/popper.min.js --output $plugdir/popper.min.js
curl https://cdnjs.cloudflare.com/ajax/libs/popper.js/$vers/umd/popper.min.js.map --output $plugdir/popper.min.js.map

# Vue 3.4.19
# https://cdnjs.com/libraries/vue
# https://unpkg.com/vue@next
vers="3.4.19"
curl https://unpkg.com/vue@$vers/dist/vue.global.js --output $plugdir/vue.global.js
curl https://unpkg.com/vue@$vers/dist/vue.global.prod.js --output $plugdir/vue.global.prod.js

# leaflet 1.9.4
# https://cdnjs.com/libraries/leaflet
vers="1.9.4"
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/$vers/leaflet.js --output $plugdir/leaflet.js
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/$vers/leaflet.js.map --output $plugdir/leaflet.js.map
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/$vers/leaflet.min.css --output $plugdir/leaflet.min.css
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/$vers/images/layers.png --output $plugdir/images/layers.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/$vers/images/layers-2x.png --output $plugdir/images/layers-2x.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/$vers/images/marker-icon.png --output $plugdir/images/marker-icon.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/$vers/images/marker-icon-2x.png --output $plugdir/images/marker-icon-2x.png
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet/$vers/images/marker-shadow.png --output $plugdir/images/marker-shadow.png

# MarkerCluster 1.5.3
# https://cdnjs.com/libraries/leaflet.markercluster
vers="1.5.3"
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/$vers/leaflet.markercluster.js --output $plugdir/leaflet.markercluster.js
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/$vers/leaflet.markercluster.js.map --output $plugdir/leaflet.markercluster.js.map
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/$vers/MarkerCluster.css --output $plugdir/MarkerCluster.css
curl https://cdnjs.cloudflare.com/ajax/libs/leaflet.markercluster/$vers/MarkerCluster.Default.css --output $plugdir/MarkerCluster.Default.css

# sha256 0.11.0
# https://cdnjs.com/libraries/js-sha256
vers="0.11.0"
curl https://cdnjs.cloudflare.com/ajax/libs/js-sha256/$vers/sha256.min.js --output $plugdir/sha256.min.js

# normalize 8.0.1
# https://cdnjs.com/libraries/normalize
vers="8.0.1"
curl https://cdnjs.cloudflare.com/ajax/libs/normalize/$vers/normalize.min.css --output $plugdir/normalize.min.css
curl https://cdnjs.cloudflare.com/ajax/libs/normalize/$vers/normalize.min.css.map --output $plugdir/normalize.min.css.map
