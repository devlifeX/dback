(function () {
  'use strict';

  if (typeof dbackDbTools === 'undefined') {
    return;
  }

  var config = dbackDbTools;

  function showStatus(elementId, message, type) {
    var node = document.getElementById(elementId);
    if (!node) {
      return;
    }

    node.className = 'notice inline notice-' + (type || 'info');
    node.textContent = message;
    node.style.display = 'block';
  }

  function hideStatus(elementId) {
    var node = document.getElementById(elementId);
    if (node) {
      node.style.display = 'none';
    }
  }

  function restUrl(path) {
    var root = String(config.restRoot || '').replace(/\/$/, '');
    return root + path;
  }

  function restHeaders(extra) {
    var headers = {
      'X-WP-Nonce': config.nonce,
    };

    if (extra) {
      Object.keys(extra).forEach(function (key) {
        headers[key] = extra[key];
      });
    }

    return headers;
  }

  function formatRestError(body) {
    if (!body) {
      return config.strings.genericError;
    }

    var message = body.message || config.strings.genericError;
    var data = body.data || {};
    var parts = [message];

    if (data.error_id) {
      parts.push('Error ID: ' + data.error_id);
    }

    if (data.operation) {
      parts.push('Operation: ' + data.operation);
    }

    if (body.code) {
      parts.push('Code: ' + body.code);
    }

    if (body.code === 'rest_no_route') {
      parts.push(config.strings.restNoRouteHint);
    }

    return parts.join(' | ');
  }

  function parseResponseError(body) {
    var error = new Error(formatRestError(body));
    error.restBody = body;
    return error;
  }

  function renderQueryResult(payload) {
    var container = document.getElementById('dback-query-result');
    if (!container) {
      return;
    }

    container.innerHTML = '';

    if (!payload || !payload.success) {
      return;
    }

    if (payload.type === 'command') {
      var summary = document.createElement('p');
      summary.textContent =
        'Affected rows: ' + String(payload.affected_rows || 0);
      container.appendChild(summary);
      return;
    }

    if (!payload.rows || !payload.rows.length) {
      var empty = document.createElement('p');
      empty.textContent = 'No rows returned.';
      container.appendChild(empty);
      return;
    }

    var table = document.createElement('table');
    table.className = 'widefat striped';

    var thead = document.createElement('thead');
    var headRow = document.createElement('tr');
    (payload.columns || Object.keys(payload.rows[0])).forEach(function (column) {
      var th = document.createElement('th');
      th.scope = 'col';
      th.textContent = column;
      headRow.appendChild(th);
    });
    thead.appendChild(headRow);
    table.appendChild(thead);

    var tbody = document.createElement('tbody');
    payload.rows.forEach(function (row) {
      var tr = document.createElement('tr');
      (payload.columns || Object.keys(row)).forEach(function (column) {
        var td = document.createElement('td');
        var value = row[column];
        td.textContent = value === null || value === undefined ? '' : String(value);
        tr.appendChild(td);
      });
      tbody.appendChild(tr);
    });
    table.appendChild(tbody);

    container.appendChild(table);
  }

  function renderErrorLog(entries) {
    var tbody = document.getElementById('dback-logs-body');
    if (!tbody) {
      return;
    }

    tbody.innerHTML = '';

    if (!entries || !entries.length) {
      var emptyRow = document.createElement('tr');
      var emptyCell = document.createElement('td');
      emptyCell.colSpan = 6;
      emptyCell.textContent = config.strings.logsEmpty;
      emptyRow.appendChild(emptyCell);
      tbody.appendChild(emptyRow);
      return;
    }

    entries.forEach(function (entry) {
      var tr = document.createElement('tr');
      var context = entry.context || {};

      [
        entry.level || '',
        entry.time || '',
        context.operation || '',
        entry.code || '',
        entry.message || '',
        entry.id || '',
      ].forEach(function (value) {
        var td = document.createElement('td');
        td.textContent = String(value);
        tr.appendChild(td);
      });

      tbody.appendChild(tr);
    });
  }

  function loadErrorLog() {
    showStatus('dback-logs-status', config.strings.logsLoading, 'info');

    return fetch(restUrl('/logs?limit=50'), {
      method: 'GET',
      headers: restHeaders(),
      credentials: 'same-origin',
    })
      .then(function (response) {
        return response.json().then(function (body) {
          if (!response.ok) {
            throw parseResponseError(body);
          }
          return body;
        });
      })
      .then(function (body) {
        renderErrorLog(body.entries || []);
        showStatus('dback-logs-status', config.strings.logsLoaded, 'success');
      })
      .catch(function (error) {
        showStatus(
          'dback-logs-status',
          error.message || config.strings.genericError,
          'error'
        );
      });
  }

  function clearErrorLog() {
    fetch(restUrl('/logs'), {
      method: 'DELETE',
      headers: restHeaders(),
      credentials: 'same-origin',
    })
      .then(function (response) {
        return response.json().then(function (body) {
          if (!response.ok) {
            throw parseResponseError(body);
          }
          return body;
        });
      })
      .then(function () {
        renderErrorLog([]);
        showStatus('dback-logs-status', config.strings.logsCleared, 'success');
      })
      .catch(function (error) {
        showStatus(
          'dback-logs-status',
          error.message || config.strings.genericError,
          'error'
        );
      });
  }

  function handleExport() {
    showStatus('dback-export-status', config.strings.exportStarted, 'info');

    fetch(restUrl('/export'), {
      method: 'GET',
      headers: restHeaders(),
      credentials: 'same-origin',
    })
      .then(function (response) {
        if (!response.ok) {
          return response.json().then(function (body) {
            throw parseResponseError(body);
          });
        }

        return response.blob().then(function (blob) {
          var url = window.URL.createObjectURL(blob);
          var link = document.createElement('a');
          link.href = url;
          link.download = 'dback-export-' + new Date().toISOString().replace(/[:.]/g, '-') + '.sql.gz';
          document.body.appendChild(link);
          link.click();
          link.remove();
          window.URL.revokeObjectURL(url);
        });
      })
      .then(function () {
        showStatus('dback-export-status', config.strings.exportDone, 'success');
      })
      .catch(function (error) {
        showStatus('dback-export-status', error.message || config.strings.genericError, 'error');
        loadErrorLog();
      });
  }

  function handleImport() {
    var fileInput = document.getElementById('dback-import-file');
    if (!fileInput || !fileInput.files || !fileInput.files.length) {
      showStatus('dback-import-status', config.strings.fileRequired, 'error');
      return;
    }

    var file = fileInput.files[0];
    showStatus('dback-import-status', config.strings.importStarted, 'info');

    fetch(restUrl('/import'), {
      method: 'POST',
      headers: restHeaders({
        'Content-Type': 'application/gzip',
      }),
      credentials: 'same-origin',
      body: file,
    })
      .then(function (response) {
        return response.json().then(function (body) {
          if (!response.ok) {
            throw parseResponseError(body);
          }
          return body;
        });
      })
      .then(function (body) {
        var message = config.strings.importDone;
        if (body && typeof body.statements_executed !== 'undefined') {
          message += ' (' + body.statements_executed + ' statements)';
        }
        showStatus('dback-import-status', message, 'success');
      })
      .catch(function (error) {
        showStatus('dback-import-status', error.message || config.strings.genericError, 'error');
        loadErrorLog();
      });
  }

  function handleQuery() {
    var sqlInput = document.getElementById('dback-query-sql');
    var sql = sqlInput ? sqlInput.value.trim() : '';

    if (!sql) {
      showStatus('dback-query-status', config.strings.sqlRequired, 'error');
      return;
    }

    hideStatus('dback-query-status');
    renderQueryResult(null);
    showStatus('dback-query-status', config.strings.queryRunning, 'info');

    fetch(restUrl('/query'), {
      method: 'POST',
      headers: restHeaders({
        'Content-Type': 'application/json',
      }),
      credentials: 'same-origin',
      body: JSON.stringify({ sql: sql }),
    })
      .then(function (response) {
        return response.json().then(function (body) {
          if (!response.ok) {
            throw parseResponseError(body);
          }
          return body;
        });
      })
      .then(function (body) {
        showStatus('dback-query-status', config.strings.queryDone, 'success');
        renderQueryResult(body);
      })
      .catch(function (error) {
        showStatus('dback-query-status', error.message || config.strings.genericError, 'error');
        loadErrorLog();
      });
  }

  document.addEventListener('DOMContentLoaded', function () {
    var exportButton = document.getElementById('dback-export-button');
    var importButton = document.getElementById('dback-import-button');
    var queryButton = document.getElementById('dback-query-button');
    var logsRefreshButton = document.getElementById('dback-logs-refresh');
    var logsClearButton = document.getElementById('dback-logs-clear');

    if (exportButton) {
      exportButton.addEventListener('click', handleExport);
    }

    if (importButton) {
      importButton.addEventListener('click', handleImport);
    }

    if (queryButton) {
      queryButton.addEventListener('click', handleQuery);
    }

    if (logsRefreshButton) {
      logsRefreshButton.addEventListener('click', loadErrorLog);
    }

    if (logsClearButton) {
      logsClearButton.addEventListener('click', clearErrorLog);
    }

    loadErrorLog();
  });
})();
