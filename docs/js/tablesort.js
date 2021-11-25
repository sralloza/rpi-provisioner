document$.subscribe(function () {
    var tables = document.querySelectorAll("article table");
    tables.forEach(function (table) {
      const classes =
        table.firstElementChild.firstElementChild.lastElementChild.classList;
      if (classes.contains("sortable")) new Tablesort(table);
    });
  });
