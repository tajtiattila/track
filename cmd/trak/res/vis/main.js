// When the window has finished loading create our google map below
google.maps.event.addDomListener(window, 'load', init);

function lat2merc(lat) {
    return 180.0 / Math.PI * Math.log(Math.tan(Math.PI/4 + lat * Math.PI/180.0/2.0));
}

function init() {
  var xv = newVisualizer(
      initMap(document.getElementById('map')),
      document.getElementById('track'));

  /*
  var vis = new TrackVisualizer(initMap(mapElement));

  getJSON("api/appdata.json", function(d) {
    vis.setDate(d.startDate);
    vis.showTrack(d.track);
  })
  */
}

function TrackVisualizer(map) {
  this.showTrack = showTrack;
  this.setDate = setDate;

  var form = new TrackForm();

  var ptListNode = document.getElementById('track');

  var locations; // track points from server
  var selIdx = null; // idx of selected

  ptListNode.onclick = clickTrackPt;
  document.addEventListener("keydown", keyDown, false);

  function setDate(d) {
    form.setDate(d);
  }

  form.onchange = newTrack;
  function newTrack(newq) {
    getJSON("api/track/"+newq, function(d) {
      showTrack(d);
    });
  }

  function showTrack(td) {
    var b = td.bounds;
    locations = td.locations;

    map.fitBounds({
      north: b.lat1,
      south: b.lat0,
      west: b.lon0,
      east: b.lon1
    });

    pointMarker(null);

    // clean old pts
    while (ptListNode.firstChild) {
        ptListNode.removeChild(ptListNode.firstChild);
    }

    // show track
    var trkpt = [];
    for (var i = 0; i < locations.length; i++) {
      var p = locations[i];
      trkpt.push({lat: p.lat, lng: p.lon})
      var div = document.createElement("div");
      div.className = "trkpt";
      div.id = "trkpt" + i.toString();
      var ts = timeStr(new Date(p.timestamp));
      ts = span(ts);
      ts.className = "timestamp";
      div.appendChild(ts);
      if (p.acc) {
        var acc = span(p.acc);
        acc.className = "accuracy";
        div.appendChild(acc);
      }
      ptListNode.appendChild(div);
    }
    showTrackLine(trkpt);
  }

  var shownTrackLine;
  function showTrackLine(pts) {
    if (shownTrackLine) {
      shownTrackLine.setMap(null);
    }
    if (!pts || pts.length === 0) {
      shownTrackLine = null;
      return;
    }

    shownTrackLine = new google.maps.Polyline({
      path: pts,
      strokeColor: "#0000FF",
      strokeOpacity: 0.4,
      strokeWidth: 2,

      map: map
    });
  }

  var shownMarker; // marker of selected element
  function pointMarker(pt) {
    if (shownMarker) {
      shownMarker.setMap(null);
    }
    if (!pt) {
      shownMarker = null;
      return;
    }
    shownMarker = new google.maps.Circle({
      center: {
        lat: pt.position.lat,
        lng: pt.position.lon,
      },
      radius: pt.accuracy,

      strokeColor: "#FF0000",
      strokeOpacity: 0.8,
      fillColor: "#FF0000",
      fillOpacity: 0.2,

      map: map
    });
  }

  function span(txt) {
    var s = document.createElement("span");
    s.innerHTML = txt;
    return s;
  }

  function scrollIntoView(elem, scroll, anim) {
    var et = $(elem).offset().top;
    var st = $(scroll).offset().top;

    var eh = $(elem).height();
    var sh = $(scroll).height();

    var margin = Math.max(2*eh, Math.floor(sh/eh/3)); // px

    var d = et - margin - st;
    if (d >= 0) {
      d = et + eh + margin - (st + sh);
      if (d <= 0) {
        return;
      }
    }

    if (anim) {
      $(scroll).animate({
        scrollTop: $(scroll).scrollTop() + d
      }, 100);
    } else {
      $(scroll).scrollTop($(scroll).scrollTop() + d);
    }
  }

  function selTrackPt(i, anim) {
    selIdx = i;
    var p = locations[i];
    $("#track").children().removeClass("selected");

    var el = document.getElementById("trkpt" + i.toString());
    $(el).addClass("selected");
    scrollIntoView(el, document.getElementById("sidebar-content"), anim);

    var acc = p.acc;
    if (!acc) {
      acc = 10e3;
    }
    pointMarker({
      position: p,
      accuracy: acc,
    });
  }

  function clickTrackPt(evt) {
    // strip trkpt part
    var t = evt.target;
    while (t && !t.id.startsWith("trkpt")) {
      t = t.parentNode;
    }
    if (t == null) {
      console.log("hopp");
      return;
    }
    var i = parseInt(t.id.slice(5));
    selTrackPt(i, true);
  }

  function keyDown(evt) {
    var i;
    if (evt.key == "j") {
      i = selIdx+1;
    }
    if (evt.key == "k") {
      i = selIdx-1;
    }
    if (0 <= i && i <= locations.length) {
      selTrackPt(i);
    }
  }
}

function initMap(mapElement) {
  // Basic options for a simple Google Map
  // For more options see: https://developers.google.com/maps/documentation/javascript/reference#MapOptions
  var mapOptions = {
      minZoom: 3,
      scaleControl: true,

      // How you would like to style the map. 
      // This is where you would paste any style found on Snazzy Maps.
      styles: [{"featureType":"landscape","stylers":[{"saturation":-100},{"lightness":65},{"visibility":"on"}]},{"featureType":"poi","stylers":[{"saturation":-100},{"lightness":51},{"visibility":"simplified"}]},{"featureType":"road.highway","stylers":[{"saturation":-100},{"visibility":"simplified"}]},{"featureType":"road.arterial","stylers":[{"saturation":-100},{"lightness":30},{"visibility":"on"}]},{"featureType":"road.local","stylers":[{"saturation":-100},{"lightness":40},{"visibility":"on"}]},{"featureType":"transit","stylers":[{"saturation":-100},{"visibility":"simplified"}]},{"featureType":"administrative.province","stylers":[{"visibility":"off"}]},{"featureType":"water","elementType":"labels","stylers":[{"visibility":"on"},{"lightness":-25},{"saturation":-100}]},{"featureType":"water","elementType":"geometry","stylers":[{"hue":"#ffff00"},{"lightness":-25},{"saturation":-97}]}]
  };

  // Create the Google Map using our element and options defined above
  return new google.maps.Map(mapElement, mapOptions);
}

function TrackForm() {
  this.onchange = function(newq) { };

  var self = this;

  var datein = document.getElementById("date-input");
  var accin = document.getElementById("accuracy-input");

  this.setDate = function(str) {
    datein.value = str;
  };

  datein.addEventListener("keydown", submitEnter);
  accin.addEventListener("keydown", submitEnter);
  document.getElementById("track-form").onsubmit = submit;

  function submitEnter(e) {
    if (!e) { var e = window.event; }

    if (e.keyCode == 13) {
      e.preventDefault(); // sometimes useful
      submit();
    }
  }

  function submit() {
    var d = new Date(datein.value);
    if (!d) return;

    var q = dateStr(d) + ".json";
    if (accin.value !== "") {
      q += "?accuracy=" + accin.value.toString();
    }
    self.onchange(q);
  }
}

function padn(num, len) {
  var pad = "0000000000";
  var s = num.toString();
  if (s.length > len)
    return s;
  return (pad + s).slice(-len);
}

function dateStr(d) {
  var y = d.getFullYear();
  var m = d.getMonth() + 1;
  var d = d.getDate();
  return padn(y, 4) + "-" + padn(m, 2) + "-" + padn(d, 2);
}

function timeStr(d, utc) {
  var h, m;
  if (utc) {
    h = d.getUTCHours();
    m = d.getUTCMinutes();
  } else {
    h = d.getHours();
    m = d.getMinutes();
  }
  var s = d.getSeconds();
  var ms = d.getMilliseconds();
  return padn(h, 2) + ":" + padn(m, 2) + ":" + padn(s, 2) + "." + padn(ms, 3);
}

var entityMap = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
  '/': '&#x2F;',
  '`': '&#x60;',
  '=': '&#x3D;'
};

function escapeHtml (string) {
  return String(string).replace(/[&<>"'`=\/]/g, function (s) {
    return entityMap[s];
  });
}

function getJSON(url, success) {
    var xhr = new XMLHttpRequest();
    xhr.onreadystatechange = function() {
      if (xhr.readyState == 4) {
        if (xhr.status == 200) {
          success(JSON.parse(xhr.responseText));
        } else {
          console.error(xhr.statusText);
        }
      }
    };
    xhr.open("GET", url, true);
    xhr.send();
}
