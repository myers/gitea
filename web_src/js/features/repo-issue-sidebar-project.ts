import {POST} from '../modules/fetch.ts';
import {IssueSidebarComboList} from './repo-issue-sidebar-combolist.ts';
import {html, htmlRaw} from '../utils/html.ts';
import {createElementFromHTML} from '../utils/dom.ts';
import $ from 'jquery';

type ColumnInfo = {
  id: number;
  title: string;
};

export function initProjectColumnPicker() {
  const columnSection = document.querySelector<HTMLElement>('#sidebar-project-column');
  if (!columnSection) return;

  initColumnDropdown(columnSection);

  const projectCombo = document.querySelector<HTMLElement>('.issue-sidebar-combo[data-update-url*="/issues/projects?"]');
  if (!projectCombo) return;

  const comboList = (projectCombo as any)._comboList as IssueSidebarComboList | undefined;
  if (!comboList) return;

  comboList.onAfterUpdate = async (response: Response, _changedValues: string[]): Promise<boolean> => {
    const data = await response.json();
    const columns: ColumnInfo[] = data.columns || [];
    const selectedColumnID: number = data.selected_column_id || 0;

    comboList.updateUiList(comboList.collectCheckedValues());

    const updateUrl = columnSection.getAttribute('data-update-url') || '';
    renderColumnPicker(columnSection, columns, selectedColumnID, updateUrl);
    return true;
  };
}

function initColumnDropdown(section: HTMLElement) {
  const dropdown = section.querySelector<HTMLElement>('.column-selector-dropdown');
  if (!dropdown) return;
  setupFomanticDropdown(dropdown);
}

function setupFomanticDropdown(el: HTMLElement) {
  const updateUrl = el.getAttribute('data-update-url');
  if (!updateUrl) return;

  $(el).dropdown({
    onChange(value: string) {
      POST(updateUrl, {data: new URLSearchParams({id: value})});
    },
  });
}

function renderColumnPicker(
  section: HTMLElement,
  columns: ColumnInfo[],
  selectedColumnID: number,
  baseUpdateUrl: string,
) {
  section.innerHTML = '';

  if (columns.length < 2) return;

  const selectedCol = columns.find((c) => c.id === selectedColumnID);
  const selectedTitle = selectedCol ? selectedCol.title : columns[0].title;

  const svgTriangle = document.querySelector('.svg.octicon-triangle-down')?.cloneNode(true) as SVGElement | null;
  const triangleHtml = svgTriangle ? (() => {
    svgTriangle.setAttribute('width', '14');
    svgTriangle.setAttribute('height', '14');
    svgTriangle.classList.add('dropdown', 'icon');
    return svgTriangle.outerHTML;
  })() : '';

  let menuItemsHtml = '';
  for (const col of columns) {
    const selectedClass = col.id === selectedColumnID ? ' active selected' : '';
    menuItemsHtml += html`<div class="item${htmlRaw(selectedClass)}" data-value="${col.id}">${col.title}</div>`;
  }

  const dropdown = createElementFromHTML(html`
    <div class="ui dropdown selection fluid column-selector-dropdown"
      data-update-url="${baseUpdateUrl}">
      <input type="hidden" name="column_id" value="${selectedColumnID}">
      <div class="default text">${selectedTitle}</div>
      ${htmlRaw(triangleHtml)}
      <div class="menu">${htmlRaw(menuItemsHtml)}</div>
    </div>
  `);

  section.append(dropdown);
  setupFomanticDropdown(dropdown);
}
