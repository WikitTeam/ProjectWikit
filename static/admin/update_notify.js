(function () {
  var STATUS_URL = '/-/admin/update/status';
  var PAGE_URL = '/-/admin/update';

  function addDot(link) {
    if (!link || link.querySelector('.wu-dot')) return;
    var dot = document.createElement('span');
    dot.className = 'wu-dot';
    dot.style.cssText = 'display:inline-block;width:8px;height:8px;border-radius:50%;background:#dc3545;margin-left:6px;vertical-align:middle;';
    link.appendChild(dot);
  }

  function findSidebarLink() {
    var links = document.querySelectorAll('#jazzy-sidebar a[href], .main-sidebar a[href], nav a[href]');
    if (!links.length) links = document.querySelectorAll('a[href]');
    for (var i = 0; i < links.length; i++) {
      var h = links[i].getAttribute('href') || '';
      if (h.indexOf('systemupdate') !== -1) return links[i];
    }
    for (var j = 0; j < links.length; j++) {
      var h2 = links[j].getAttribute('href') || '';
      if (h2.indexOf('/-/admin/update') !== -1 &&
          h2.indexOf('/status') === -1 && h2.indexOf('/trigger') === -1) {
        return links[j];
      }
    }
    return null;
  }

  function showToast(tag) {
    var key = 'wu_dismiss_' + (tag || '');
    if (localStorage.getItem(key)) return;
    if (document.getElementById('wu-toast')) return;

    var box = document.createElement('div');
    box.id = 'wu-toast';
    box.style.cssText = 'position:fixed;right:20px;bottom:20px;z-index:99999;background:#fff;border:1px solid #ddd;border-left:4px solid #dc3545;border-radius:8px;box-shadow:0 4px 16px rgba(0,0,0,.15);padding:14px 16px;max-width:320px;font-size:14px;color:#333;';
    box.innerHTML =
      '<div style="font-weight:600;margin-bottom:6px;">有新版本可用' + (tag ? '：' + tag : '') + '</div>' +
      '<div style="margin-bottom:10px;color:#666;">可在「系统更新」页面查看并更新。</div>' +
      '<div style="text-align:right;">' +
        '<a id="wu-toast-view" href="' + PAGE_URL + '" style="margin-right:12px;color:#1967d2;text-decoration:none;">查看</a>' +
        '<a id="wu-toast-dismiss" href="#" style="color:#999;text-decoration:none;">不再提醒</a>' +
      '</div>';
    document.body.appendChild(box);

    document.getElementById('wu-toast-dismiss').addEventListener('click', function (e) {
      e.preventDefault();
      localStorage.setItem(key, '1');
      if (box.parentNode) box.parentNode.removeChild(box);
    });
  }

  function check() {
    fetch(STATUS_URL, { credentials: 'same-origin' })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (d) {
        if (!d || !d.available) return;
        addDot(findSidebarLink());
        showToast((d.latest && d.latest.tag) || '');
      })
      .catch(function () {});
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', check);
  } else {
    check();
  }
})();
