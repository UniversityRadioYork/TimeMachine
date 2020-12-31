var startTime;
var showStartTime;
var showEndTime;
var startTimeEpoch;

var xmlhttp = new XMLHttpRequest();
xmlhttp.open('GET', 'https://ury.org.uk/api/v2/timeslot/currentandnext?api_key=' + window.APIKey, true);
xmlhttp.onreadystatechange = function() {
    if (xmlhttp.readyState == 4) {
        if(xmlhttp.status == 200) {
            var obj = JSON.parse(xmlhttp.responseText);
            var payload = obj.payload;
            var currentShow = payload.current;

            // If the current show has no start time (jukebox/off air), allow seek to beginning of hour.
            if (!currentShow.start_time) {
              var start_time = new Date();
              start_time.setHours(start_time.getHours() - 1)
              start_time.setMinutes(0);
              start_time.setSeconds(0);
              currentShow.start_time = start_time.getTime() / 1000;
            }
            if (isNaN(currentShow.end_time)) {
              var end_time = new Date();
              end_time.setHours(end_time.getHours() + 1)
              end_time.setMinutes(0);
              end_time.setSeconds(0);
              currentShow.end_time = end_time.getTime() / 1000;
            }

            startTimeEpoch = 447045;//Math.round(currentShow.start_time / (60*60));


            const secondsSinceEpoch = Math.round(Date.now() / 1000)
            const delta = secondsSinceEpoch - currentShow.start_time
            console.log(delta)

            document.title = currentShow.title + " - URY Rewind"

            audio.setAttribute("title", currentShow.title + " - URY Rewinder")

            showStartTime = new Date(currentShow.start_time * 1000)
            showEndTime = new Date(currentShow.end_time * 1000)
            $(".show-name").text(currentShow.title)
            $(".show-date").text(showStartTime.toDateString())
            $(".show-starttime").text(showStartTime.toLocaleTimeString())
            $(".show-endtime").text(showEndTime.toLocaleTimeString())
            $(".show-image").attr("src", "https://ury.org.uk" + currentShow.photo);
            seekToProgress(0, false)


            console.log(currentShow)
        }
    }
};
xmlhttp.send(null);

var timeout;
var dragging;

var showDuration = function() {
  return showEndTime.getTime() - showStartTime.getTime()
}
var showElapsedPercent = function() {
  var now = new Date();

  return ((now.getTime() - showStartTime.getTime()) / showDuration()) * 100
}

var player = document.getElementById('audio');
player.addEventListener("timeupdate", function() {

  if (startTime && !dragging) {
    $(".playback-time").text(playerTime().toLocaleTimeString())


  var percent_through_show = showElapsedPercent()

  $("#audio-progress").css(
    {
      'width': percent_through_show + "%"
    }
  );

  var now = new Date();
  var percent = (playerTime().getTime() - showStartTime.getTime()) / (now.getTime() - showStartTime.getTime())*100
    $('#audio-progress-bar').css({
      'width': percent + "%"
    });
    $('#draggable-point').css({
      'left': percent*percent_through_show/100 + "%"
    });
  }
});

var playerTime = function(offset) {
  if (startTime) {
    var currentPlaybackTime
    var currentTime
    if (offset) {
      currentTime = offset
      currentPlaybackTime = new Date(showStartTime.getTime() + currentTime*1000);
    } else {
      currentTime = player.currentTime;
      currentPlaybackTime = new Date(startTime.getTime() + currentTime*1000);
    }

    return currentPlaybackTime
  }
}

var seekToProgress = function(offset, startPlaying = true) {
  changePlayState(false)
  dragging = false;
  offset = Math.round(offset);
  console.log("Seeking to offset:", offset)

  var audio = document.getElementById('audio');

  var source = document.getElementById('audioSource');
  source.src = "tm/v1/show/" + startTimeEpoch + "/stream?offset=" + offset

  audio.load(); //call this to just preload the audio without playing

  if (startPlaying) {
    changePlayState(true); //call this to play the song right away
  }

  startTime = new Date(showStartTime.getTime() + offset*1000);

}
var playState;

var pButton = "#show-player-play"; // Play/Pause/Stop button

function changePlayState(state) {
  //Only modify audio player if we're making a change.
  if (state) {
    player.play();
    $(pButton).removeClass("fa-play").removeClass("fa-close").addClass("fa-pause");
  } else {
    player.pause();
    $(pButton).removeClass("fa-pause").removeClass("fa-close").addClass("fa-play");
  }
  playState = state;
};

$(pButton).click(function(e) {
  changePlayState(!playState);
});

$('#audio-progress').click(function(e) {
  var elm = $(this);
  var xPos = (e.pageX - elm.offset().left) / elm.width();

  var percent_through_show = showElapsedPercent()

  //console.log(xPos)

  var requested_percent = xPos*percent_through_show;
  console.log("Requested %", requested_percent)
  offset = requested_percent/100*(showDuration()/1000)
  console.log(offset)
  seekToProgress(offset)
})

$('#draggable-point').draggable({
  axis: 'x',
  containment: "#audio-progress"
});

$('#draggable-point').draggable({
  drag: function() {
    dragging = true;
    var offset = $(this).offset();
    var xPos = (100 * parseFloat($(this).css("left"))) / (parseFloat($(this).parent().css("width")))

    var percent_through_show = showElapsedPercent()

    //console.log(xPos)

    var requested_percent = xPos*percent_through_show/100
    console.log("Requested percent", requested_percent)
    //offset = percent_through_show/100*showDuration()/1000 - requested_percent/100 * showDuration()/1000
    offset = requested_percent/100*(showDuration()/1000)

    //console.log("percent through show", percent_through_show)

    //console.log("handle offset", handle_offset)

    //console.log("Behind by", offset)
    $(".playback-time").text(playerTime(offset).toLocaleTimeString());
    if (timeout) {
      clearTimeout(timeout)
    }
    timeout = setTimeout(() => {seekToProgress(offset)}, 300)
  }
});
