var x = fetch('http://www.example.com/').then(function(r) {
  r.text().then(function(d) {
    console.log(r.statusText);

    for (var k in r.headers._headers) {
      console.log(k + ':', r.headers.get(k));
    }
    console.log('');

    console.log(d);
  });
});
