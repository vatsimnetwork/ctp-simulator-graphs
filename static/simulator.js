(function () {
  /* ── Theme toggle ── */
  var themeBtn = document.getElementById('theme-toggle');
  if (themeBtn) {
    themeBtn.addEventListener('click', function () {
      var isDark = !document.documentElement.classList.contains('dark');
      document.documentElement.classList.toggle('dark', isDark);
      localStorage.setItem('theme', isDark ? 'dark' : 'light');
      rebuildOpenCharts();
    });
  }

  /* ── Table sorting ── */
  document.querySelectorAll('.data-table').forEach(function (table) {
    var headers = table.querySelectorAll('th[data-sort]');
    headers.forEach(function (th, colIdx) {
      th.addEventListener('click', function () {
        var tbody = table.querySelector('tbody');
        if (!tbody) return;

        var rows = Array.from(tbody.querySelectorAll('tr.data-row'));
        var dir = th.dataset.dir === 'asc' ? 'desc' : 'asc';

        headers.forEach(function (h) { delete h.dataset.dir; });
        th.dataset.dir = dir;

        var sortKey = th.dataset.sort;
        rows.sort(function (a, b) {
          var aVal = a.querySelector('[data-col="' + sortKey + '"]');
          var bVal = b.querySelector('[data-col="' + sortKey + '"]');
          if (!aVal || !bVal) return 0;
          var aText = aVal.dataset.value || aVal.textContent.trim();
          var bText = bVal.dataset.value || bVal.textContent.trim();
          var aNum = parseFloat(aText);
          var bNum = parseFloat(bText);
          var cmp;
          if (!isNaN(aNum) && !isNaN(bNum)) {
            cmp = aNum - bNum;
          } else {
            cmp = aText.localeCompare(bText);
          }
          return dir === 'asc' ? cmp : -cmp;
        });

        rows.forEach(function (row) {
          var chartRow = row.nextElementSibling;
          tbody.appendChild(row);
          if (chartRow && chartRow.classList.contains('chart-row')) {
            tbody.appendChild(chartRow);
          }
        });
      });
    });
  });

  /* ── Table filtering ── */
  document.querySelectorAll('.filter-input[data-filter-table]').forEach(function (input) {
    var tableId = input.dataset.filterTable;
    input.addEventListener('input', function () {
      var pattern = input.value.trim().toUpperCase();
      var table = document.getElementById(tableId);
      if (!table) return;
      var rows = table.querySelectorAll('tbody tr.data-row');
      var visible = 0;
      rows.forEach(function (row) {
        var text = row.textContent.toUpperCase();
        var match = !pattern || text.indexOf(pattern) !== -1;
        row.style.display = match ? '' : 'none';
        var chartRow = row.nextElementSibling;
        if (chartRow && chartRow.classList.contains('chart-row')) {
          chartRow.style.display = match && chartRow.classList.contains('open') ? '' : 'none';
        }
        if (match) visible++;
      });
      var countEl = document.getElementById(tableId + '-count');
      if (countEl) {
        countEl.textContent = visible + ' of ' + rows.length;
      }
    });
  });

  /* ── Expandable chart rows ── */
  var openCharts = {};

  function barColor(pct) {
    if (pct >= 100) return 'rgba(220,38,38,0.75)';
    if (pct >= 80)  return 'rgba(194,97,42,0.75)';
    return 'rgba(22,163,74,0.75)';
  }

  function buildRowChart(canvas, series, labels, maxSlots) {
    var isDark = document.documentElement.classList.contains('dark');
    var gridColor = isDark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.06)';
    var tickColor = isDark ? '#a1a1aa' : '#64748b';
    var refColor  = isDark ? 'rgba(220,38,38,0.35)' : 'rgba(220,38,38,0.30)';

    var colors = series.map(function (v) {
      if (maxSlots > 0) return barColor((v / maxSlots) * 100);
      return 'rgba(22,163,74,0.75)';
    });

    var datasets = [{
      data: series,
      backgroundColor: colors,
      borderRadius: 2,
      borderSkipped: false,
      order: 2
    }];

    if (maxSlots > 0) {
      datasets.push({
        type: 'line',
        data: series.map(function () { return maxSlots; }),
        borderColor: refColor,
        borderWidth: 1,
        borderDash: [4, 3],
        pointRadius: 0,
        fill: false,
        order: 1
      });
    }

    return new Chart(canvas, {
      type: 'bar',
      data: { labels: labels, datasets: datasets },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            filter: function (item) { return item.datasetIndex === 0; },
            callbacks: {
              label: function (ctx) { return ctx.parsed.y + ' slots'; }
            }
          }
        },
        scales: {
          x: {
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 10 }, maxRotation: 0 }
          },
          y: {
            min: 0,
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 10 } }
          }
        }
      }
    });
  }

  function buildRowLineChart(canvas, series, labels) {
    var isDark = document.documentElement.classList.contains('dark');
    var gridColor = isDark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.06)';
    var tickColor = isDark ? '#a1a1aa' : '#64748b';

    return new Chart(canvas, {
      type: 'line',
      data: {
        labels: labels,
        datasets: [{
          data: series,
          borderColor: 'rgba(59,130,246,0.8)',
          backgroundColor: 'rgba(59,130,246,0.1)',
          borderWidth: 2,
          pointRadius: 0,
          tension: 0.3,
          fill: true
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            mode: 'index',
            intersect: false,
            callbacks: {
              label: function (ctx) { return ctx.parsed.y + ' slots'; }
            }
          }
        },
        scales: {
          x: {
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 9 }, maxRotation: 0, autoSkip: true, maxTicksLimit: 20 }
          },
          y: {
            min: 0,
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 10 } }
          }
        }
      }
    });
  }

  function rebuildOpenCharts() {
    Object.keys(openCharts).forEach(function (canvasId) {
      var entry = openCharts[canvasId];
      if (entry.chart) entry.chart.destroy();
      var canvas = document.getElementById(canvasId);
      if (canvas) {
        entry.chart = buildRowChart(canvas, entry.series, entry.labels, entry.maxSlots);
      }
    });
  }

  document.querySelectorAll('.expand-row').forEach(function (row) {
    row.addEventListener('click', function () {
      var chartRowId = row.dataset.chartRow;
      var chartRow = document.getElementById(chartRowId);
      if (!chartRow) return;

      var isOpen = chartRow.classList.contains('open');
      if (isOpen) {
        chartRow.classList.remove('open');
        row.classList.remove('expanded');
        var canvasId = chartRow.querySelector('canvas').id;
        if (openCharts[canvasId]) {
          openCharts[canvasId].chart.destroy();
          delete openCharts[canvasId];
        }
      } else {
        chartRow.classList.add('open');
        row.classList.add('expanded');
        var canvas = chartRow.querySelector('canvas');
        if (canvas && canvas.dataset.series) {
          var series = JSON.parse(canvas.dataset.series);
          var labels = JSON.parse(canvas.dataset.labels);
          var maxSlots = parseInt(canvas.dataset.max || '0', 10);
          var chart = buildRowChart(canvas, series, labels, maxSlots);
          openCharts[canvas.id] = { chart: chart, series: series, labels: labels, maxSlots: maxSlots };
        }
      }
    });
  });

  /* ── Timing page: facility selector ── */
  var enrouteSelector = document.getElementById('enroute-facility-selector');
  if (enrouteSelector) {
    enrouteSelector.addEventListener('change', function () {
      document.querySelectorAll('.enroute-group').forEach(function (g) {
        g.style.display = 'none';
      });
      var selected = enrouteSelector.value;
      if (selected) {
        var group = document.getElementById('enroute-' + selected);
        if (group) group.style.display = '';
      }
    });
    enrouteSelector.dispatchEvent(new Event('change'));
  }

  /* ── Auto-init timing charts ── */
  document.querySelectorAll('.auto-chart').forEach(function (canvas) {
    if (canvas.dataset.series) {
      var series = JSON.parse(canvas.dataset.series);
      var labels = JSON.parse(canvas.dataset.labels);
      var maxSlots = parseInt(canvas.dataset.max || '0', 10);
      buildRowChart(canvas, series, labels, maxSlots);
    }
  });

  /* ── Departure airports grid ── */
  var depGrid = document.getElementById('dep-airport-grid');
  var depDetail = document.getElementById('dep-detail');
  var depDetailTitle = document.getElementById('dep-detail-title');
  var depTbody = document.getElementById('dep-slot-tbody');
  var activeDepCard = null;

  if (depGrid) {
    depGrid.querySelectorAll('.dep-airport-card').forEach(function (card) {
      card.addEventListener('click', function () {
        if (activeDepCard === card) {
          card.classList.remove('active');
          depDetail.style.display = 'none';
          activeDepCard = null;
          return;
        }
        if (activeDepCard) activeDepCard.classList.remove('active');
        card.classList.add('active');
        activeDepCard = card;

        var name = card.dataset.name;
        var slots = JSON.parse(card.dataset.slots || '[]');
        depDetailTitle.textContent = name + ' — ' + slots.length + ' slot' + (slots.length !== 1 ? 's' : '');

        depTbody.innerHTML = '';
        slots.forEach(function (s) {
          var tr = document.createElement('tr');
          tr.innerHTML = '<td>' + s.dep + '</td><td>' + s.arr + '</td><td>' + s.depTime + '</td><td>' + s.arrTime + '</td>';
          depTbody.appendChild(tr);
        });

        depDetail.style.display = '';
        depDetail.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      });
    });
  }

  /* ── Sectors grid ── */
  var sectorGrid = document.getElementById('sector-grid');
  var sectorDetail = document.getElementById('sector-detail');
  var sectorDetailTitle = document.getElementById('sector-detail-title');
  var sectorCanvas = document.getElementById('sector-chart-canvas');
  var sectorChartWrap = document.getElementById('sector-chart-wrap');
  var sectorNoTimings = document.getElementById('sector-no-timings');
  var sectorFineBtn = document.getElementById('sector-fine-btn');
  var sectorFineLoading = document.getElementById('sector-fine-loading');
  var activeSectorCard = null;
  var sectorChart = null;
  var sectorFineMode = false;
  var sectorFineData = null;

  if (sectorGrid) {
    sectorGrid.querySelectorAll('.sector-card').forEach(function (card) {
      card.addEventListener('click', function () {
        if (activeSectorCard === card) {
          card.classList.remove('active');
          sectorDetail.style.display = 'none';
          activeSectorCard = null;
          if (sectorChart) { sectorChart.destroy(); sectorChart = null; }
          return;
        }
        if (activeSectorCard) activeSectorCard.classList.remove('active');
        card.classList.add('active');
        activeSectorCard = card;

        var name = card.dataset.name;
        var hasTimings = card.dataset.hasTimings !== 'false';

        sectorDetailTitle.textContent = name;
        sectorDetail.style.display = '';
        sectorFineMode = false;
        sectorFineData = null;
        if (sectorFineBtn) {
          sectorFineBtn.style.display = hasTimings ? '' : 'none';
          sectorFineBtn.disabled = false;
          sectorFineBtn.textContent = 'Load 2-min data';
        }
        if (sectorFineLoading) sectorFineLoading.style.display = 'none';

        if (sectorChart) { sectorChart.destroy(); sectorChart = null; }

        if (!hasTimings) {
          sectorNoTimings.style.display = '';
          sectorChartWrap.style.display = 'none';
        } else {
          sectorNoTimings.style.display = 'none';
          sectorChartWrap.style.display = '';
          var series = JSON.parse(card.dataset.series || '[]');
          var labels = JSON.parse(card.dataset.labels || '[]');
          var maxAcph = parseInt(card.dataset.max || '0', 10);
          sectorChart = buildRowChart(sectorCanvas, series, labels, maxAcph);
        }


      });
    });
  }

  if (sectorFineBtn) {
    sectorFineBtn.addEventListener('click', function () {
      if (!activeSectorCard) return;
      var eventId = new URLSearchParams(window.location.search).get('event');
      var sectorId = activeSectorCard.dataset.name;
      if (!eventId || !sectorId) return;

      sectorFineBtn.disabled = true;
      sectorFineBtn.textContent = 'Loading...';
      if (sectorFineLoading) sectorFineLoading.style.display = '';

      fetch(window.CTP_BASE_PATH + '/charts/sector/' + encodeURIComponent(sectorId) + '/fine?event=' + encodeURIComponent(eventId))
        .then(function (r) { return r.json(); })
        .then(function (resp) {
          sectorFineData = resp;
          sectorFineMode = true;
          if (sectorChart) { sectorChart.destroy(); sectorChart = null; }
          sectorChart = buildRowLineChart(sectorCanvas, resp.data, resp.labels);
          sectorFineBtn.style.display = 'none';
          if (sectorFineLoading) sectorFineLoading.style.display = 'none';
        })
        .catch(function () {
          sectorFineBtn.disabled = false;
          sectorFineBtn.textContent = 'Load 2-min data';
          if (sectorFineLoading) sectorFineLoading.style.display = 'none';
        });
    });
  }

  /* ── Arrival airports grid ── */
  var arrGrid = document.getElementById('arr-airport-grid');
  var arrDetail = document.getElementById('arr-detail');
  var arrDetailTitle = document.getElementById('arr-detail-title');
  var arrCanvas = document.getElementById('arr-chart-canvas');
  var arrToggleBar = document.getElementById('arr-toggle-bar');
  var arrToggleLine = document.getElementById('arr-toggle-line');
  var arrFineBtn = document.getElementById('arr-fine-btn');
  var arrFineLoading = document.getElementById('arr-fine-loading');
  var activeArrCard = null;
  var arrChart = null;
  var arrChartMode = 'bar';
  var arrCurrentData = null;
  var arrFineMode = false;
  var arrFineData = null;

  function buildArrBarChart(canvas, labels, total) {
    var isDark = document.documentElement.classList.contains('dark');
    var gridColor = isDark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.06)';
    var tickColor = isDark ? '#a1a1aa' : '#64748b';
    return new Chart(canvas, {
      type: 'bar',
      data: {
        labels: labels,
        datasets: [{
          data: total,
          backgroundColor: 'rgba(59,130,246,0.7)',
          borderRadius: 2,
          borderSkipped: false
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: { label: function (ctx) { return ctx.parsed.y + ' arrivals'; } }
          }
        },
        scales: {
          x: {
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 10 }, maxRotation: 0 }
          },
          y: {
            min: 0,
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 10 } }
          }
        }
      }
    });
  }

  function buildArrLineChart(canvas, labels, total, depSeries) {
    var isDark = document.documentElement.classList.contains('dark');
    var gridColor = isDark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.06)';
    var tickColor = isDark ? '#a1a1aa' : '#64748b';
    var totalColor = isDark ? 'rgba(255,255,255,0.5)' : 'rgba(0,0,0,0.35)';

    var datasets = depSeries.map(function (ds) {
      return {
        label: ds.dep,
        data: ds.buckets,
        borderColor: ds.color,
        backgroundColor: 'transparent',
        borderWidth: 2,
        pointRadius: 2,
        tension: 0.3,
        fill: false
      };
    });
    datasets.push({
      label: 'Total',
      data: total,
      borderColor: totalColor,
      backgroundColor: 'transparent',
      borderWidth: 2,
      borderDash: [4, 3],
      pointRadius: 0,
      tension: 0.3,
      fill: false
    });

    return new Chart(canvas, {
      type: 'line',
      data: { labels: labels, datasets: datasets },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            display: true,
            position: 'bottom',
            labels: {
              color: tickColor,
              font: { family: 'Ubuntu', size: 10 },
              boxWidth: 12,
              padding: 8
            }
          },
          tooltip: {
            mode: 'index',
            intersect: false
          }
        },
        scales: {
          x: {
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 10 }, maxRotation: 0 }
          },
          y: {
            min: 0,
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 10 } }
          }
        }
      }
    });
  }

  function buildArrLineChartFine(canvas, labels, total) {
    var isDark = document.documentElement.classList.contains('dark');
    var gridColor = isDark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.06)';
    var tickColor = isDark ? '#a1a1aa' : '#64748b';

    return new Chart(canvas, {
      type: 'line',
      data: {
        labels: labels,
        datasets: [{
          data: total,
          borderColor: 'rgba(59,130,246,0.8)',
          backgroundColor: 'rgba(59,130,246,0.1)',
          borderWidth: 2,
          pointRadius: 0,
          tension: 0.3,
          fill: true
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            mode: 'index',
            intersect: false,
            callbacks: {
              label: function (ctx) { return ctx.parsed.y + ' arrivals'; }
            }
          }
        },
        scales: {
          x: {
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 9 }, maxRotation: 0, autoSkip: true, maxTicksLimit: 30 }
          },
          y: {
            min: 0,
            grid: { color: gridColor },
            ticks: { color: tickColor, font: { family: 'Ubuntu', size: 10 } }
          }
        }
      }
    });
  }

  function renderArrChart() {
    if (!arrCanvas) return;
    if (arrFineMode && arrFineData) {
      if (arrChart) { arrChart.destroy(); arrChart = null; }
      arrChart = buildArrLineChartFine(arrCanvas, arrFineData.labels, arrFineData.total);
      return;
    }
    if (!arrCurrentData) return;
    if (arrChart) { arrChart.destroy(); arrChart = null; }
    if (arrChartMode === 'bar') {
      arrChart = buildArrBarChart(arrCanvas, arrCurrentData.labels, arrCurrentData.total);
    } else {
      arrChart = buildArrLineChart(arrCanvas, arrCurrentData.labels, arrCurrentData.total, arrCurrentData.depSeries);
    }
  }

  if (arrGrid) {
    arrGrid.querySelectorAll('.arr-airport-card').forEach(function (card) {
      card.addEventListener('click', function () {
        if (activeArrCard === card) {
          card.classList.remove('active');
          arrDetail.style.display = 'none';
          activeArrCard = null;
          if (arrChart) { arrChart.destroy(); arrChart = null; }
          arrCurrentData = null;
          arrFineMode = false;
          arrFineData = null;
          return;
        }
        if (activeArrCard) activeArrCard.classList.remove('active');
        card.classList.add('active');
        activeArrCard = card;

        arrFineMode = false;
        arrFineData = null;
        if (arrFineBtn) {
          arrFineBtn.style.display = '';
          arrFineBtn.disabled = false;
          arrFineBtn.textContent = 'Load 2-min data';
        }
        if (arrFineLoading) arrFineLoading.style.display = 'none';

        arrCurrentData = {
          name: card.dataset.name,
          labels: JSON.parse(card.dataset.labels || '[]'),
          total: JSON.parse(card.dataset.total || '[]'),
          depSeries: JSON.parse(card.dataset.depSeries || '[]')
        };
        arrDetailTitle.textContent = arrCurrentData.name;
        arrDetail.style.display = '';

        arrChartMode = 'bar';
        arrToggleBar.classList.add('active');
        arrToggleLine.classList.remove('active');
        renderArrChart();
      });
    });

    if (arrToggleBar) {
      arrToggleBar.addEventListener('click', function () {
        if (arrChartMode === 'bar' || arrFineMode) return;
        arrChartMode = 'bar';
        arrToggleBar.classList.add('active');
        arrToggleLine.classList.remove('active');
        renderArrChart();
      });
    }
    if (arrToggleLine) {
      arrToggleLine.addEventListener('click', function () {
        if (arrChartMode === 'line' || arrFineMode) return;
        arrChartMode = 'line';
        arrToggleLine.classList.add('active');
        arrToggleBar.classList.remove('active');
        renderArrChart();
      });
    }
  }

  if (arrFineBtn) {
    arrFineBtn.addEventListener('click', function () {
      if (!activeArrCard) return;
      var eventId = new URLSearchParams(window.location.search).get('event');
      var arrId = activeArrCard.dataset.name;
      if (!eventId || !arrId) return;

      arrFineBtn.disabled = true;
      arrFineBtn.textContent = 'Loading...';
      if (arrFineLoading) arrFineLoading.style.display = '';

      fetch(window.CTP_BASE_PATH + '/charts/arrival/' + encodeURIComponent(arrId) + '/fine?event=' + encodeURIComponent(eventId))
        .then(function (r) { return r.json(); })
        .then(function (resp) {
          arrFineData = resp;
          arrFineMode = true;
          arrToggleBar.classList.remove('active');
          arrToggleLine.classList.remove('active');
          if (arrChart) { arrChart.destroy(); arrChart = null; }
          arrChart = buildArrLineChartFine(arrCanvas, resp.labels, resp.total);
          arrFineBtn.style.display = 'none';
          if (arrFineLoading) arrFineLoading.style.display = 'none';
        })
        .catch(function () {
          arrFineBtn.disabled = false;
          arrFineBtn.textContent = 'Load 2-min data';
          if (arrFineLoading) arrFineLoading.style.display = 'none';
        });
    });
  }
})();
