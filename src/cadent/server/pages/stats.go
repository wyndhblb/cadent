/*
Copyright 2014-2017 Bo Blanton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pages

var STATS_INDEX_PAGE = `
<!DOCTYPE html>
<html>
<head>
	<!--

                                        +oyhdddddddddddddddddddddhyo+
                                      ohNMMMMMMMMMMMMMMMMMMMMMMMMMMMNdo
                                     yNMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMh
                                    sNMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMd
                                    dMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMo
                                    mMMMMNoosNMMMMMMMMMMMMMMMMMMdoohMMMMMo
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  yMMMMMo
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  yMMMMMs
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  yMMMMMs
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  yMMMMMs
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  yMMMMMs
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  yMMMMMo
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  sMMMMMs
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  sMMMMMs
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  sMMMMMs
                                    mMMMMN   NMMMMMMMMMMMMMMMMMMh  sMMMMMs
                                    mMMMMm   NMMMMMMMMMMMMMMMMMMh  sMMMMMs
                                    odNNmo   NMMMMMMMNmmMMMMMMMMh   ymNNh
                                             NMMMMMMMs  dMMMMMMMh
                                             NMMMMMMMs  dMMMMMMMd
                                             NMMMMMMMs  dMMMMMMMd
                                             NMMMMMMMs  dMMMMMMMd
                                             NMMMMMMMs  dMMMMMMMd
                                             NMMMMMMMs  dMMMMMMMd
                                             NMMMMMMMs  dMMMMMMMd
                                             NMMMMMMMs  dMMMMMMMd
                                             NMMMMMMMs  hMMMMMMMd
                                             NMMMMMMMs  dMMMMMMMd
                                             NMMMMMMMs  hMMMMMMMd
                                             NMMMMMMMs  hMMMMMMMd
                        +syhhhyo             NMMMMMMMs  hMMMMMMMd
                      odNMMMMMMMNd+          NMMMMMMMs  hMMMMMMMd
                     sNMMMMMMMMMMMNo         NMMMMMMMs  hMMMMMMMd
                     NMMMMMMMMMMMMMm         NMMMMMMMs  hMMMMMMMd
                    +MMMMMMMMMMMMMMm         NMMMMMMMs  hMMMMMMMd
                     dMMMMMMMMMMMMMs         NMMMMMMMs  hMMMMMMMd
                      yNMMMMMMMMMmo          hMMMMMMm+  +NMMMMMNs
                        oydmmmdy+             +yddhs      shdhy+

    -->

	<title>Cadent Tracker</title>
	<script src="//ajax.googleapis.com/ajax/libs/jquery/1.10.2/jquery.min.js"></script>
	<link rel="stylesheet" href="//netdna.bootstrapcdn.com/bootstrap/3.1.1/css/bootstrap.min.css">
	<link rel="stylesheet" href="//netdna.bootstrapcdn.com/bootstrap/3.1.1/css/bootstrap-theme.min.css">
	<script src="//netdna.bootstrapcdn.com/bootstrap/3.1.1/js/bootstrap.min.js"></script>
	<script src="//cdnjs.cloudflare.com/ajax/libs/dygraph/1.1.0/dygraph-combined.js"></script>
	<style type="text/css">
		.graph{
			height:400px;
			width:100%;
			
		}
	</style>
	
</head>
<body>

<div class="container">

	<div class="page-header">
		<div class="row">
			<div class="col-sm-12"><h1>Cadent Tracker <small>Up: <span id="uptime_sec"></span></small></h1></div>
		</div>
	</div>
	<div class="row">
		<div class="col-sm-3"><h4>All Lines: <span id="all_lines_count">Getting Stats... hold on</span></h4></div>
		<div class="col-sm-3"><h4>Valid Lines: <span id="valid_line_count"></span></h4></div>
		<div class="col-sm-3"><h4>Sent Lines: <span id="success_send_count"></span></h4></div>
		<div class="col-sm-3"><h4>Failed Lines: <span id="fail_send_count"></span></h4></div>
	</div>
	<div class="row">
		<div class="col-sm-12">
			<h4>Current Running Servers:</h4>
			<div id="current-running">
				
			</div>
		</div>
	</div>
	
	<div class="row">
		<div class="col-sm-12">
		<div>
			<h4> Lines Seen</h4>
			<div id="current_all_lines_count" class="graph"></div>
			<span id="current_all_lines_count_leg"></span>
		</div>
		<div>
			<h4>Valid Lines</h4>
			<div id="current_valid_line_count"  class="graph"></div>
			<span id="current_valid_line_count_leg"></span>
		</div>
		<div>
			<h4>InValid Lines</h4>
			<div id="current_invalid_line_count"  class="graph"></div>
			<span id="current_invalid_line_count_leg"></span>
		</div>
		<div>
			<h4>Sent Lines</h4>
			<div id="current_success_send_count"  class="graph"></div>
			<span id="current_success_send_count_leg"></span>
		</div>
		<div>
			<h4>Failed Lines</h4>
			<div id="current_fail_send_count"  class="graph"></div>
			<span id="current_fail_send_count_leg"></span>
		</div>
		<div>
			<h4>UnSendable Lines</h4>
			<div id="current_unsendable_send_count"  class="graph"></div>
			<span id="current_unsendable_send_count_leg"></span>
		</div>
		<div>
			<h4>Unknown Lines</h4>
			<div id="current_unknown_send_count" class="graph"></div>
			<span id="current_unknown_send_count_leg"></span>
		</div>
		

		</div>
		</div>

	<div class="footer small">
		<hr>
		<ul class="list-inline links pull-left">
&copy; CADENT: Bo Blanton 2015 - 2017
		</ul>

		<ul class="list-inline pull-right">
			<li class="pull-right"></li>
		</ul>

		<div class="clearfix"></div>
	</div>

</div>

<script type="text/javascript">
	function toprettytime(seconds) {
		var sec_num = parseInt(seconds, 10);
		var hours   = Math.floor(sec_num / 3600);
		var minutes = Math.floor((sec_num - (hours * 3600)) / 60);
		var seconds = sec_num - (hours * 3600) - (minutes * 60);

		if (hours   < 10) {hours   = "0"+hours;}
		if (minutes < 10) {minutes = "0"+minutes;}
		if (seconds < 10) {seconds = "0"+seconds;}
		var time    = hours+':'+minutes+':'+seconds;
		return time;
	}

	var HOST="/stats"
	var DATA = {}
	var LAST_DATA = {}
	var GRAPHS = {}
	var servers = []
	var ticks = []
	var MAX_POINTS=300
	var statInterval=5000
	var TickInterval=5 // stats every 5 second are rendered by the Cadent

	function renderGraphs(){
		now = new Date().getTime();
		$.each( GRAPHS, function( key, graph ) {
			if(GRAPHS[key]['graph'] == null){
				var init_data = {
					drawPoints: true,
					showRoller: true,
					ylabel: 'Count/s',
					legend: 'always',
					//valueRange: [0.0, 1.2],
					labels: ["Time"],
					axes: {
						x: {
							axisLabelFormatter: function(x) {
								return Dygraph.dateString_(x)
							}
						}
					}
					
				};
				data = []

				$.each(servers, function(idx, servername) {
					chart_data = LAST_DATA[servername]
					// back fill from array keys
					//[time, data, data, data]
					if (data.length == 0) {

						$.each(chart_data['ticks_list'], function (idx, time) {
							data.push([time])
						});

					}
					data_key = (key + "_list").replace("current_", "")
					$.each(chart_data[data_key], function (idx, datapoint) {
						data[idx].push(datapoint / TickInterval) //per stats are ticked every 5 seconds
					});


					init_data['labels'].push(servername)

				})
				//console.log(data)
				GRAPHS[key]['data'] = data
				GRAPHS[key]['graph'] = new Dygraph(graph['item'], GRAPHS[key]['data'], init_data);
			}else{
				t_data = []
				$.each(servers, function(idx, servername) {
					chart_data = LAST_DATA[servername]
					if(t_data.length == 0){
						t_data.push(now)
					}
					t_data.push(chart_data[key] / TickInterval)  //per stats are ticked every 5 seconds
				})
				GRAPHS[key]['data'].push(t_data)
				if(GRAPHS[key]['data'].length > MAX_POINTS){
					GRAPHS[key]['data'] = GRAPHS[key]['data'].slice(-MAX_POINTS);
				}
				//console.log(GRAPHS[key]['data'])
				GRAPHS[key]['graph'].updateOptions( { 'file': GRAPHS[key]['data'] } );
				
			}

		})

	}
	function getData(){
		$.ajax({
			url: HOST,
			jsonp: "jsonp",
			dataType: "jsonp",


			// Work with the response
			success: function( data ) {
				
				$('#current-running').html(" ")
				
				now = new Date().getTime();
				ticks.push(now)
				LAST_DATA = data
				$.each( data, function( key, val ) {
					if(!DATA[key]){
						servers.push(key)

						DATA[key] = {}
						$.each( val, function( s_k, d_k ) {
							DATA[key][s_k] = [d_k]
							DATA[key][s_k+"_time"] = [now]
							if($("#"+s_k).get(0) && $("#"+s_k).hasClass("graph")){
								GRAPHS[s_k] = {
									'item': $("#"+s_k).get(0),
									'graph' : null
								}
							}
						})

					}else{
						$.each( val, function( s_k, d_k ) {
							DATA[key][s_k].push(d_k)
							DATA[key][s_k+"_time"].push(now);
							
							if($("#"+s_k).get(0) && !$("#"+s_k).hasClass("graph")){
								if (s_k.match(/uptime/)){
									d_k = toprettytime(d_k)
								}
								$("#"+s_k).html(d_k)
							}
						})
					}
					var on_html = $('#current-running').html()
					//change
					var news = "<a onclick=\"$('#" + key + "_serv').toggle()\">" + key  + ": listening on " + LAST_DATA[key]['listening'] +"</a><div id='" + key + "_serv' style='display:none'>"
					news += "<ul><li><strong>UP: </strong> " + (LAST_DATA[key]['servers_up']?LAST_DATA[key]['servers_up'].join(", "):"None") + "</li>";
					news += "<li><strong>CHECKING: </strong> " +  (LAST_DATA[key]['servers_checking']?LAST_DATA[key]['servers_checking'].join(", "):"None") + "</li>";
					news += "<li><strong>DOWN: </strong> " + (LAST_DATA[key]['servers_down']?LAST_DATA[key]['servers_down'].join(", "):"None") + "</li>";
					news += "</ul></div><hr />"
					$('#current-running').html(on_html + news)

				});
				//console.log(DATA)
				renderGraphs()
			}
		});

	}
	setInterval(getData, statInterval)
</script>

</body>
</html>
`
