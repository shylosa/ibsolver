export function renderPreparationTable(container, table) {
    const fragment = document.createDocumentFragment();

    const scroll = document.createElement('div');
    scroll.className = 'reference-table-scroll';
    const element = document.createElement('table');
    element.className = 'reference-table';
    element.dataset.tableId = table.id;
    const caption = document.createElement('caption');
    caption.className = 'visually-hidden';
    caption.textContent = table.title;
    element.append(caption);
    const columnGroup = document.createElement('colgroup');
    for (const _column of table.columns) columnGroup.append(document.createElement('col'));
    element.append(columnGroup);

    const head = document.createElement('thead');
    const headingRow = document.createElement('tr');
    for (const column of table.columns) {
        if (column.headerHidden) continue;
        const cell = document.createElement('th');
        cell.scope = 'col';
        if (column.headerColSpan) cell.colSpan = column.headerColSpan;
        const label = document.createElement('span');
        label.className = 'reference-table-label';
        label.textContent = column.label;
        cell.append(label);
        if (column.unit) {
            const unit = document.createElement('span');
            unit.className = 'reference-table-unit';
            unit.textContent = column.unit;
            cell.append(unit);
        }
        headingRow.append(cell);
    }
    head.append(headingRow);
    element.append(head);

    const body = document.createElement('tbody');
    for (const row of table.rows) {
        const rowElement = document.createElement('tr');
        const mergedCells = new Map((row.mergedCells || []).map(merge => [merge.column, merge.colSpan]));
        for (let index = 0; index < row.cells.length; index++) {
            const value = row.cells[index];
            const colSpan = mergedCells.get(index) || 1;
            const cell = document.createElement('td');
            const column = table.columns[index];
            cell.colSpan = colSpan;
            cell.className = column.text ? 'reference-table-text' : 'reference-table-value';
            if (value === '') cell.setAttribute('aria-label', 'Порожня клітинка');
            else cell.textContent = value;
            rowElement.append(cell);
            index += colSpan - 1;
        }
        if (table.groupColumn && row.group) {
            const group = document.createElement('td');
            group.className = 'reference-table-text reference-table-group';
            group.rowSpan = row.group.rowSpan;
            group.textContent = row.group.label;
            rowElement.append(group);
        }
        body.append(rowElement);
    }
    element.append(body);
    scroll.append(element);
    fragment.append(scroll);
    container.replaceChildren(fragment);
}
