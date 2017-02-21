

function getCurrent() {
$.ajax({
  url: "/api/get",
  type: 'get',
  dataType: 'json',
  success: function (data) {
    /*
    type IPStatus struct {
      IP string `json:"ipaddr"`
      IsFree bool `json:"isfree"`
    }
    */
    
    var newHTML = '<table class="table"><tr><th>IP Address</th><th>Status</th></tr>';
    for (var prop in data) {
      console.log(JSON.stringify(data[prop]));
      if (data.hasOwnProperty(prop)) {
        if (data[prop].isfree) {
          newHTML += '<tr class="success"><td>' + data[prop].ipaddr + '</td><td>Available</td></tr>';
        }
      }
    }
    newHTML += "</table>";
    $('#IPTABLESTATUS').html(newHTML);
  },
  error: function(data) {
      alert('ERROR: ' + data.ErrorMessage);
  }});
}