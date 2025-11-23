import Quill from 'quill';
import 'quill/dist/quill.snow.css';
import QuillBetterTable from 'quill-better-table';
import 'quill-better-table/dist/quill-better-table.css';
import './style.css';

const API_BASE = 'http://localhost:8080/api';

interface Entry {
  id: string;
  user_id: string;
  title: string;
  body_delta: any;
  body_html: string;
  attendees_original: string;
  attendees: string[];
  type: 'meeting' | 'notes' | 'other';
  day_year: number;
  day_month: number;
  day_day: number;
  created_at: string;
  updated_at: string;
}

class JournalApp {
  private currentDate: Date;
  private selectedDate: Date;
  private entries: Entry[] = [];
  private daysWithEntries: Set<number> = new Set();
  private currentEditingEntry: Entry | null = null;
  private quill: Quill | null = null;
  private autoSaveTimer: number | null = null;
  private searchQuery: string = '';
  private chatCitations: Map<string, Entry[]> = new Map(); // messageId -> cited entries
  private attendeesAutocomplete: HTMLElement | null = null;
  private autocompleteTimeout: number | null = null;

  constructor() {
    this.currentDate = new Date();
    this.selectedDate = new Date();
    this.init();
  }

  private async init() {
    // Check if configuration is required
    const configValid = await this.checkConfiguration();
    if (!configValid) {
      this.showConfigSetup();
      return;
    }

    this.renderCalendar();
    this.loadDaysWithEntries();
    this.loadEntries();
    this.setupEventListeners();
    this.createSearchPanel();
    this.createChatPanel();
  }

  private setupEventListeners() {
    document.getElementById('prev-month')?.addEventListener('click', () => {
      this.currentDate.setMonth(this.currentDate.getMonth() - 1);
      this.renderCalendar();
      this.loadDaysWithEntries();
    });

    document.getElementById('next-month')?.addEventListener('click', () => {
      this.currentDate.setMonth(this.currentDate.getMonth() + 1);
      this.renderCalendar();
      this.loadDaysWithEntries();
    });

    document.getElementById('new-entry-btn')?.addEventListener('click', () => {
      this.createNewEntry();
    });

    document.getElementById('journal-nav')?.addEventListener('click', () => {
      this.switchToCalendarView();
    });

    document.getElementById('search-nav')?.addEventListener('click', () => {
      this.switchToSearchView();
    });

    document.getElementById('chat-nav')?.addEventListener('click', () => {
      this.switchToChatView();
    });

    document.getElementById('settings-nav')?.addEventListener('click', () => {
      this.showConfigSetup(true);
    });
  }

  private createSearchPanel() {
    const leftPanel = document.getElementById('left-panel');
    if (!leftPanel) return;

    const searchPanel = document.createElement('aside');
    searchPanel.className = 'search-panel hidden';
    searchPanel.id = 'search-panel';
    searchPanel.innerHTML = `
      <div class="search-header">
        <h2>Search</h2>
        <input type="text" class="search-input" id="search-input" placeholder="Search entries...">
      </div>
      <div id="search-results-info"></div>
    `;

    leftPanel.parentElement?.insertBefore(searchPanel, leftPanel);

    const searchInput = document.getElementById('search-input') as HTMLInputElement;
    if (searchInput) {
      let searchTimeout: number;
      searchInput.addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        searchTimeout = window.setTimeout(() => {
          this.searchQuery = (e.target as HTMLInputElement).value;
          this.performSearch();
        }, 300);
      });
    }
  }

  private switchToCalendarView() {
    document.getElementById('left-panel')?.classList.remove('hidden');
    document.getElementById('search-panel')?.classList.add('hidden');
    document.getElementById('chat-panel')?.classList.add('hidden');
    document.getElementById('journal-nav')?.classList.add('active');
    document.getElementById('search-nav')?.classList.remove('active');
    document.getElementById('chat-nav')?.classList.remove('active');
    
    // Show entries panel and reload current day's entries
    const entriesPanel = document.querySelector('.entries-panel') as HTMLElement;
    if (entriesPanel) {
      entriesPanel.style.display = 'flex';
    }
    
    // Reload current day's entries (this will also update the header)
    this.loadEntries();
  }

  private switchToSearchView() {
    document.getElementById('left-panel')?.classList.add('hidden');
    document.getElementById('search-panel')?.classList.remove('hidden');
    document.getElementById('chat-panel')?.classList.add('hidden');
    document.getElementById('journal-nav')?.classList.remove('active');
    document.getElementById('search-nav')?.classList.add('active');
    document.getElementById('chat-nav')?.classList.remove('active');
    
    // Show entries panel
    const entriesPanel = document.querySelector('.entries-panel') as HTMLElement;
    if (entriesPanel) {
      entriesPanel.style.display = 'flex';
    }
    
    // Update header
    const header = document.getElementById('selected-date');
    if (header) {
      header.textContent = 'Search Results';
    }

    // If there's a search query, perform search, otherwise clear entries
    if (this.searchQuery.trim()) {
      this.performSearch();
    } else {
      this.entries = [];
      this.renderEntries();
      const infoEl = document.getElementById('search-results-info');
      if (infoEl) {
        infoEl.textContent = '';
      }
    }

    // Focus search input
    setTimeout(() => {
      document.getElementById('search-input')?.focus();
    }, 100);
  }

  private switchToChatView() {
    document.getElementById('left-panel')?.classList.add('hidden');
    document.getElementById('search-panel')?.classList.add('hidden');
    document.getElementById('chat-panel')?.classList.remove('hidden');
    document.getElementById('journal-nav')?.classList.remove('active');
    document.getElementById('search-nav')?.classList.remove('active');
    document.getElementById('chat-nav')?.classList.add('active');
    
    // Show entries panel for displaying source entries
    const entriesPanel = document.querySelector('.entries-panel') as HTMLElement;
    if (entriesPanel) {
      entriesPanel.style.display = 'flex';
    }
    
    // Update header
    const header = document.getElementById('selected-date');
    if (header) {
      header.textContent = 'AI Assistant';
    }
    
    // Clear entries initially (they'll be populated when chat responds with sources)
    this.entries = [];
    this.renderEntries();

    // Focus chat input
    setTimeout(() => {
      document.getElementById('chat-input')?.focus();
    }, 100);
  }

  private createChatPanel() {
    const leftPanel = document.getElementById('left-panel');
    if (!leftPanel) return;

    const chatPanel = document.createElement('aside');
    chatPanel.className = 'chat-panel hidden';
    chatPanel.id = 'chat-panel';
    chatPanel.innerHTML = `
      <div class="chat-header">
        <h2>AI Assistant</h2>
      </div>
      <div class="chat-messages" id="chat-messages">
        <div class="chat-message assistant">
          <div class="chat-message-content">
            Hi! I can help you find and understand your journal entries. Ask me anything about your notes, meetings, or experiences.
          </div>
          <div class="chat-message-time">Just now</div>
        </div>
      </div>
      <div class="chat-input-container">
        <div class="chat-input-wrapper">
          <textarea id="chat-input" class="chat-input" placeholder="Ask about your journal..." rows="1"></textarea>
          <button id="chat-send-btn" class="chat-send-btn">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="22" y1="2" x2="11" y2="13"></line>
              <polygon points="22 2 15 22 11 13 2 9 22 2"></polygon>
            </svg>
          </button>
        </div>
      </div>
    `;

    // Insert chat panel in same position as other left panels
    const entriesPanel = document.querySelector('.entries-panel');
    entriesPanel?.parentElement?.insertBefore(chatPanel, entriesPanel);

    // Setup chat input handlers
    const chatInput = document.getElementById('chat-input') as HTMLTextAreaElement;
    const chatSendBtn = document.getElementById('chat-send-btn') as HTMLButtonElement;

    if (chatInput) {
      // Auto-resize textarea
      chatInput.addEventListener('input', () => {
        chatInput.style.height = 'auto';
        chatInput.style.height = Math.min(chatInput.scrollHeight, 100) + 'px';
      });

      // Send on Enter (Shift+Enter for newline)
      chatInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
          e.preventDefault();
          this.sendChatMessage();
        }
      });
    }

    if (chatSendBtn) {
      chatSendBtn.addEventListener('click', () => {
        this.sendChatMessage();
      });
    }
  }

  private async sendChatMessage() {
    const chatInput = document.getElementById('chat-input') as HTMLTextAreaElement;
    const chatMessages = document.getElementById('chat-messages');
    const sendBtn = document.getElementById('chat-send-btn') as HTMLButtonElement;

    if (!chatInput || !chatMessages) return;

    const message = chatInput.value.trim();
    if (!message) return;

    // Disable input while processing
    chatInput.disabled = true;
    sendBtn.disabled = true;

    // Add user message to chat
    const userMessageEl = document.createElement('div');
    userMessageEl.className = 'chat-message user';
    userMessageEl.innerHTML = `
      <div class="chat-message-content">${this.escapeHtml(message)}</div>
      <div class="chat-message-time">${new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</div>
    `;
    chatMessages.appendChild(userMessageEl);

    // Clear input
    chatInput.value = '';
    chatInput.style.height = 'auto';

    // Add typing indicator
    const typingEl = document.createElement('div');
    typingEl.className = 'chat-message assistant';
    typingEl.innerHTML = `
      <div class="chat-typing">
        <div class="chat-typing-dot"></div>
        <div class="chat-typing-dot"></div>
        <div class="chat-typing-dot"></div>
      </div>
    `;
    chatMessages.appendChild(typingEl);
    chatMessages.scrollTop = chatMessages.scrollHeight;

    try {
      const response = await fetch(`${API_BASE}/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message })
      });

      // Remove typing indicator
      typingEl.remove();

      if (response.ok) {
        const data = await response.json();
        
        // Store citations for this message
        if (data.message_id && data.source_entries) {
          this.chatCitations.set(data.message_id, data.source_entries);
        }
        
        // Display source entries in the entries panel if available
        if (data.source_entries && data.source_entries.length > 0) {
          this.entries = data.source_entries;
          this.renderEntries();
          
          // Show entries panel
          const entriesPanel = document.querySelector('.entries-panel') as HTMLElement;
          if (entriesPanel) {
            entriesPanel.style.display = 'flex';
          }
          
          // Update header to show source count
          const header = document.getElementById('selected-date');
          if (header) {
            header.textContent = `AI Assistant - ${data.source_entries.length} source ${data.source_entries.length === 1 ? 'entry' : 'entries'}`;
          }
        }
        
        // Add assistant response
        const assistantMessageEl = document.createElement('div');
        assistantMessageEl.className = 'chat-message assistant';
        assistantMessageEl.dataset.messageId = data.message_id;
        assistantMessageEl.innerHTML = `
          <div class="chat-message-content">${this.escapeHtml(data.response)}</div>
          <div class="chat-message-time">${new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</div>
        `;
        
        // Add click handler to show citations
        assistantMessageEl.addEventListener('click', () => {
          const messageId = assistantMessageEl.dataset.messageId;
          if (messageId && this.chatCitations.has(messageId)) {
            const citations = this.chatCitations.get(messageId)!;
            this.entries = citations;
            this.renderEntries();
            
            // Update header
            const header = document.getElementById('selected-date');
            if (header) {
              header.textContent = `AI Assistant - ${citations.length} source ${citations.length === 1 ? 'entry' : 'entries'}`;
            }
            
            // Add visual feedback
            document.querySelectorAll('.chat-message.assistant').forEach(el => {
              el.classList.remove('active-citations');
            });
            assistantMessageEl.classList.add('active-citations');
          }
        });
        
        chatMessages.appendChild(assistantMessageEl);
      } else {
        // Show error message
        const errorMessageEl = document.createElement('div');
        errorMessageEl.className = 'chat-message assistant';
        errorMessageEl.innerHTML = `
          <div class="chat-message-content" style="color: #d32f2f;">Sorry, I encountered an error. Please try again.</div>
          <div class="chat-message-time">${new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</div>
        `;
        chatMessages.appendChild(errorMessageEl);
      }
    } catch (error) {
      console.error('Chat error:', error);
      typingEl.remove();
      
      const errorMessageEl = document.createElement('div');
      errorMessageEl.className = 'chat-message assistant';
      errorMessageEl.innerHTML = `
        <div class="chat-message-content" style="color: #d32f2f;">Sorry, I couldn't connect to the AI service.</div>
        <div class="chat-message-time">${new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</div>
      `;
      chatMessages.appendChild(errorMessageEl);
    } finally {
      // Re-enable input
      chatInput.disabled = false;
      sendBtn.disabled = false;
      chatInput.focus();
      chatMessages.scrollTop = chatMessages.scrollHeight;
    }
  }

  private escapeHtml(text: string): string {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  private async performSearch() {
    if (!this.searchQuery.trim()) {
      this.entries = [];
      this.renderEntries();
      const infoEl = document.getElementById('search-results-info');
      if (infoEl) {
        infoEl.textContent = '';
      }
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/search?q=${encodeURIComponent(this.searchQuery)}`);
      if (response.ok) {
        this.entries = await response.json();
        this.renderEntries();
        
        const infoEl = document.getElementById('search-results-info');
        if (infoEl) {
          infoEl.innerHTML = `<p style="font-size: 12px; color: #666; margin-top: 12px;">${this.entries.length} result${this.entries.length !== 1 ? 's' : ''} found</p>`;
        }
      }
    } catch (error) {
      console.error('Search failed:', error);
    }
  }

  private renderCalendar() {
    const year = this.currentDate.getFullYear();
    const month = this.currentDate.getMonth();

    const monthYearEl = document.getElementById('month-year');
    if (monthYearEl) {
      monthYearEl.textContent = new Date(year, month).toLocaleDateString('en-US', {
        month: 'long',
        year: 'numeric'
      });
    }

    const calendar = document.getElementById('calendar');
    if (!calendar) return;

    calendar.innerHTML = '';

    // Day headers
    const dayHeaders = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    dayHeaders.forEach(day => {
      const header = document.createElement('div');
      header.className = 'calendar-day-header';
      header.textContent = day;
      calendar.appendChild(header);
    });

    // Calculate first day of month and total days
    const firstDay = new Date(year, month, 1).getDay();
    const daysInMonth = new Date(year, month + 1, 0).getDate();
    const prevMonthDays = new Date(year, month, 0).getDate();

    // Previous month days
    for (let i = firstDay - 1; i >= 0; i--) {
      const day = prevMonthDays - i;
      const dayEl = this.createDayElement(day, true, year, month - 1);
      calendar.appendChild(dayEl);
    }

    // Current month days
    for (let day = 1; day <= daysInMonth; day++) {
      const dayEl = this.createDayElement(day, false, year, month);
      calendar.appendChild(dayEl);
    }

    // Next month days
    const remainingDays = 42 - (firstDay + daysInMonth);
    for (let day = 1; day <= remainingDays; day++) {
      const dayEl = this.createDayElement(day, true, year, month + 1);
      calendar.appendChild(dayEl);
    }
  }

  private createDayElement(day: number, isOtherMonth: boolean, year: number, month: number): HTMLElement {
    const dayEl = document.createElement('div');
    dayEl.className = 'calendar-day';
    dayEl.textContent = day.toString();

    if (isOtherMonth) {
      dayEl.classList.add('other-month');
    }

    const date = new Date(year, month, day);
    const today = new Date();
    
    if (this.isSameDay(date, today)) {
      dayEl.classList.add('today');
    }

    if (this.isSameDay(date, this.selectedDate)) {
      dayEl.classList.add('selected');
    }

    if (!isOtherMonth && this.daysWithEntries.has(day)) {
      dayEl.classList.add('has-entries');
    }

    dayEl.addEventListener('click', () => {
      this.selectedDate = new Date(year, month, day);
      if (month !== this.currentDate.getMonth()) {
        this.currentDate = new Date(year, month, day);
        this.renderCalendar();
        this.loadDaysWithEntries();
      } else {
        this.renderCalendar();
      }
      this.loadEntries();
    });

    return dayEl;
  }

  private isSameDay(d1: Date, d2: Date): boolean {
    return d1.getFullYear() === d2.getFullYear() &&
           d1.getMonth() === d2.getMonth() &&
           d1.getDate() === d2.getDate();
  }

  private async loadDaysWithEntries() {
    const year = this.currentDate.getFullYear();
    const month = this.currentDate.getMonth() + 1;

    try {
      const response = await fetch(`${API_BASE}/months/${year}-${String(month).padStart(2, '0')}/entry-days`);
      const data = await response.json();
      this.daysWithEntries = new Set(data.daysWithEntries);
      this.renderCalendar();
    } catch (error) {
      console.error('Failed to load days with entries:', error);
    }
  }

  private async loadEntries() {
    const year = this.selectedDate.getFullYear();
    const month = this.selectedDate.getMonth() + 1;
    const day = this.selectedDate.getDate();

    const selectedDateEl = document.getElementById('selected-date');
    if (selectedDateEl) {
      selectedDateEl.textContent = this.selectedDate.toLocaleDateString('en-US', {
        weekday: 'long',
        year: 'numeric',
        month: 'long',
        day: 'numeric'
      });
    }

    try {
      const response = await fetch(`${API_BASE}/days/${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}/entries`);
      this.entries = await response.json();
      this.renderEntries();
    } catch (error) {
      console.error('Failed to load entries:', error);
      this.entries = [];
      this.renderEntries();
    }
  }

  private renderEntries() {
    const container = document.getElementById('entries-container');
    if (!container) return;

    container.innerHTML = '';

    if (this.entries.length === 0) {
      container.innerHTML = '<div class="empty-state">No entries for this day. Click "+ New Entry" to create one.</div>';
      return;
    }

    this.entries.forEach(entry => {
      const card = this.createEntryCard(entry);
      container.appendChild(card);
    });
  }

  private createEntryCard(entry: Entry): HTMLElement {
    const card = document.createElement('div');
    card.className = 'entry-card';
    card.dataset.entryId = entry.id;

    const header = document.createElement('div');
    header.className = 'entry-header';

    const meta = document.createElement('div');
    meta.className = 'entry-meta';

    const title = document.createElement('div');
    title.className = 'entry-title-display';
    title.textContent = entry.title || '(Untitled)';

    const info = document.createElement('div');
    info.className = 'entry-info';
    
    const typeSpan = `<span class="entry-type">${entry.type}</span>`;
    const attendeesSpan = entry.attendees.length > 0 
      ? `<span class="entry-attendees">with ${entry.attendees.join(', ')}</span>` 
      : '';
    
    info.innerHTML = typeSpan + (attendeesSpan ? ' ' + attendeesSpan : '');

    meta.appendChild(title);
    meta.appendChild(info);

    header.appendChild(meta);

    const body = document.createElement('div');
    body.className = 'entry-body-display';
    body.innerHTML = entry.body_html || '<em>No content</em>';

    card.appendChild(header);
    card.appendChild(body);

    card.addEventListener('click', () => {
      if (!card.classList.contains('editing')) {
        this.editEntry(entry, card);
      }
    });

    return card;
  }

  private async createNewEntry() {
    const year = this.selectedDate.getFullYear();
    const month = this.selectedDate.getMonth() + 1;
    const day = this.selectedDate.getDate();
    const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;

    const newEntry = {
      title: '',
      body_delta: { ops: [{ insert: '\n' }] },
      body_html: '<p><br></p>',
      body_text: '',
      attendees_original: '',
      type: 'notes',
      date: dateStr
    };

    try {
      const response = await fetch(`${API_BASE}/entries`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newEntry)
      });

      if (response.ok) {
        const entry = await response.json();
        this.entries.unshift(entry);
        this.renderEntries();
        this.daysWithEntries.add(day);
        this.renderCalendar();
        
        // Edit the new entry
        const card = document.querySelector(`[data-entry-id="${entry.id}"]`) as HTMLElement;
        if (card) {
          this.editEntry(entry, card);
        }
      }
    } catch (error) {
      console.error('Failed to create entry:', error);
    }
  }

  private editEntry(entry: Entry, card: HTMLElement) {
    // Close any existing editor
    if (this.currentEditingEntry) {
      this.saveEntry();
    }

    this.currentEditingEntry = entry;
    card.classList.add('editing');

    // Clear card
    card.innerHTML = '';

    // Create edit form
    const header = document.createElement('div');
    header.className = 'entry-header';

    const titleInput = document.createElement('input');
    titleInput.type = 'text';
    titleInput.className = 'entry-title-input';
    titleInput.value = entry.title;
    titleInput.placeholder = 'Entry title...';

    header.appendChild(titleInput);
    card.appendChild(header);

    // Quill editor
    const editorContainer = document.createElement('div');
    editorContainer.id = 'quill-editor';
    card.appendChild(editorContainer);

    // Register table module
    Quill.register('modules/better-table', QuillBetterTable);

    const quillInstance = new Quill(editorContainer, {
      theme: 'snow',
      modules: {
        toolbar: {
          container: [
            ['bold', 'italic', 'underline'],
            [{ 'list': 'ordered'}, { 'list': 'bullet' }],
            [{ 'header': [1, 2, 3, false] }],
            ['insertTable'],
            ['clean']
          ],
          handlers: {
            'insertTable': function(this: any) {
              const tableModule = this.quill.getModule('better-table');
              tableModule.insertTable(3, 3);
            }
          }
        },
        'better-table': {
          operationMenu: {
            items: {
              unmergeCells: {
                text: 'Unmerge cells'
              }
            }
          }
        },
        keyboard: {
          bindings: QuillBetterTable.keyboardBindings
        }
      }
    });

    this.quill = quillInstance;

    this.quill.setContents(entry.body_delta);

    // Form fields
    const formFields = document.createElement('div');
    formFields.className = 'entry-form-fields';

    const typeField = document.createElement('div');
    typeField.className = 'form-field';
    typeField.innerHTML = `
      <label>Type</label>
      <select id="entry-type">
        <option value="notes" ${entry.type === 'notes' ? 'selected' : ''}>Notes</option>
        <option value="meeting" ${entry.type === 'meeting' ? 'selected' : ''}>Meeting</option>
        <option value="other" ${entry.type === 'other' ? 'selected' : ''}>Other</option>
      </select>
    `;

    const attendeesField = document.createElement('div');
    attendeesField.className = 'form-field';
    attendeesField.innerHTML = `
      <label>Attendees (comma-separated)</label>
      <input type="text" id="entry-attendees" value="${entry.attendees_original}" placeholder="Alice, Bob, Carol" autocomplete="off">
    `;

    formFields.appendChild(typeField);
    formFields.appendChild(attendeesField);
    card.appendChild(formFields);

    // Setup autocomplete for attendees
    const attendeesInput = attendeesField.querySelector('#entry-attendees') as HTMLInputElement;
    if (attendeesInput) {
      this.setupAttendeesAutocomplete(attendeesInput);
    }

    // Actions
    const actions = document.createElement('div');
    actions.className = 'form-actions';

    const deleteBtn = document.createElement('button');
    deleteBtn.className = 'delete-btn';
    deleteBtn.textContent = 'Delete';
    deleteBtn.onclick = (e) => {
      e.stopPropagation();
      this.deleteEntry(entry.id);
    };

    const saveBtn = document.createElement('button');
    saveBtn.className = 'save-btn';
    saveBtn.textContent = 'Save';
    saveBtn.onclick = () => this.saveEntry();

    const cancelBtn = document.createElement('button');
    cancelBtn.className = 'cancel-btn';
    cancelBtn.textContent = 'Cancel';
    cancelBtn.onclick = () => this.cancelEdit();

    actions.appendChild(deleteBtn);
    actions.appendChild(cancelBtn);
    actions.appendChild(saveBtn);
    card.appendChild(actions);

    // Auto-save on blur
    titleInput.addEventListener('blur', () => this.scheduleAutoSave());
    this.quill.on('text-change', () => this.scheduleAutoSave());
    document.getElementById('entry-type')?.addEventListener('change', () => this.scheduleAutoSave());
    document.getElementById('entry-attendees')?.addEventListener('blur', () => this.scheduleAutoSave());

    // Focus title
    titleInput.focus();
  }

  private scheduleAutoSave() {
    if (this.autoSaveTimer) {
      clearTimeout(this.autoSaveTimer);
    }
    this.autoSaveTimer = window.setTimeout(() => {
      this.saveEntry(true);
    }, 2000);
  }

  private async saveEntry(silent = false) {
    if (!this.currentEditingEntry || !this.quill) return;

    const card = document.querySelector(`[data-entry-id="${this.currentEditingEntry.id}"]`) as HTMLElement;
    if (!card) return;

    const titleInput = card.querySelector('.entry-title-input') as HTMLInputElement;
    const typeSelect = card.querySelector('#entry-type') as HTMLSelectElement;
    const attendeesInput = card.querySelector('#entry-attendees') as HTMLInputElement;

    const updateData = {
      title: titleInput?.value || '',
      body_delta: this.quill.getContents(),
      body_html: this.quill.root.innerHTML,
      body_text: this.quill.getText(),
      type: typeSelect?.value || 'notes',
      attendees_original: attendeesInput?.value || ''
    };

    try {
      const response = await fetch(`${API_BASE}/entries/${this.currentEditingEntry.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updateData)
      });

      if (response.ok) {
        const updatedEntry = await response.json();
        const index = this.entries.findIndex(e => e.id === updatedEntry.id);
        if (index !== -1) {
          this.entries[index] = updatedEntry;
        }
        
        if (!silent) {
          this.currentEditingEntry = null;
          this.quill = null;
          this.renderEntries();
        }
      }
    } catch (error) {
      console.error('Failed to save entry:', error);
    }
  }

  private cancelEdit() {
    this.currentEditingEntry = null;
    this.quill = null;
    if (this.autoSaveTimer) {
      clearTimeout(this.autoSaveTimer);
    }
    this.renderEntries();
  }

  private async deleteEntry(id: string) {
    if (!confirm('Are you sure you want to delete this entry?')) return;

    try {
      const response = await fetch(`${API_BASE}/entries/${id}`, {
        method: 'DELETE'
      });

      if (response.ok || response.status === 204) {
        this.entries = this.entries.filter(e => e.id !== id);
        this.renderEntries();
        
        // Update days with entries
        if (this.entries.length === 0) {
          this.daysWithEntries.delete(this.selectedDate.getDate());
          this.renderCalendar();
        }
      }
    } catch (error) {
      console.error('Failed to delete entry:', error);
    }
  }

  private async checkConfiguration(): Promise<boolean> {
    try {
      // Try to make a simple API call to check if the backend is properly configured
      const year = new Date().getFullYear();
      const month = new Date().getMonth() + 1;
      const day = new Date().getDate();
      const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
      
      const response = await fetch(`${API_BASE}/days/${dateStr}/entries`);
      
      // If we get a success response, configuration is valid
      if (response.ok) {
        return true;
      }
      
      // If we get 503, configuration is required
      if (response.status === 503) {
        return false;
      }
      
      // If we get a server error, likely database connection issue
      // which means we need configuration
      if (response.status >= 500) {
        return false;
      }
      
      // For other errors (4xx), assume configuration is OK
      return true;
    } catch (error) {
      // Network error or server not running with full config - need setup
      console.log('Configuration check failed:', error);
      return false;
    }
  }

  private showConfigSetup(showCancel: boolean = false) {
    const app = document.getElementById('app');
    if (!app) return;

    // If showing cancel, append overlay instead of replacing content
    const overlayHTML = `
      <div class="config-overlay">
        <div class="config-modal">
          <h2>🔧 ${showCancel ? 'Settings' : 'Initial Setup Required'}</h2>
          <p>Please configure your database and AI settings to continue.</p>
          
          <form id="config-form">
            <div class="config-section">
              <h3>Database Configuration</h3>
              <div class="form-row">
                <label>
                  Host:
                  <input type="text" name="database_host" value="localhost" required>
                </label>
                <label>
                  Port:
                  <input type="text" name="database_port" value="5432" required>
                </label>
              </div>
              <div class="form-row">
                <label>
                  Database Name:
                  <input type="text" name="database_name" value="journal" required>
                </label>
              </div>
              <div class="form-row">
                <label>
                  Username:
                  <input type="text" name="database_user" value="journal" required>
                </label>
                <label>
                  Password:
                  <input type="password" name="database_password" value="journaldev" required>
                </label>
              </div>
              <div class="form-row">
                <label>
                  SSL Mode:
                  <select name="database_ssl_mode">
                    <option value="disable" selected>Disable</option>
                    <option value="require">Require</option>
                    <option value="verify-ca">Verify CA</option>
                    <option value="verify-full">Verify Full</option>
                  </select>
                </label>
              </div>
            </div>

            <div class="config-section">
              <h3>AI Configuration</h3>
              <div class="form-row">
                <label>
                  Ollama URL:
                  <input type="text" name="ollama_base_url" value="http://localhost:11434" required>
                </label>
              </div>
              <div class="form-row">
                <label>
                  Embedding Model:
                  <input type="text" name="embedding_model" value="nomic-embed-text" required>
                </label>
                <label>
                  Chat Model:
                  <input type="text" name="chat_model" value="llama3.2" required>
                </label>
              </div>
            </div>

            <div class="config-actions">
              <button type="submit" class="btn-primary">Save Configuration</button>
              ${showCancel ? '<button type="button" class="btn-secondary" id="config-cancel">Cancel</button>' : ''}
            </div>

            ${showCancel ? `
              <div class="config-section" style="margin-top: 32px; border-top: 1px solid #e0e0e0; padding-top: 32px;">
                <h3>Export Data</h3>
                <p style="color: #666; margin-bottom: 16px;">Download all your journal entries as a ZIP file.</p>
                <button type="button" class="btn-secondary" id="export-entries">📦 Export All Entries</button>
              </div>
            ` : ''}

            <div id="config-error" class="config-error hidden"></div>
            <div id="config-success" class="config-success hidden"></div>
          </form>
        </div>
      </div>
    `;

    if (showCancel) {
      // Append as overlay when opened from settings
      app.insertAdjacentHTML('beforeend', overlayHTML);
    } else {
      // Replace entire content for initial setup
      app.innerHTML = overlayHTML;
    }

    const form = document.getElementById('config-form') as HTMLFormElement;
    form?.addEventListener('submit', async (e) => {
      e.preventDefault();
      await this.saveConfiguration(new FormData(form));
    });

    if (showCancel) {
      document.getElementById('config-cancel')?.addEventListener('click', () => {
        const overlay = document.querySelector('.config-overlay');
        overlay?.remove();
      });

      document.getElementById('export-entries')?.addEventListener('click', async () => {
        try {
          const response = await fetch(`${API_BASE}/export`);
          
          if (!response.ok) {
            throw new Error('Export failed');
          }

          // Get the blob from the response
          const blob = await response.blob();
          
          // Create a download link
          const url = window.URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = `journal_export_${new Date().toISOString().split('T')[0]}.zip`;
          document.body.appendChild(a);
          a.click();
          
          // Cleanup
          window.URL.revokeObjectURL(url);
          document.body.removeChild(a);
          
          // Show success message
          const successEl = document.getElementById('config-success');
          if (successEl) {
            successEl.textContent = '✅ Export downloaded successfully!';
            successEl.classList.remove('hidden');
            setTimeout(() => {
              successEl.classList.add('hidden');
            }, 3000);
          }
        } catch (error) {
          console.error('Export error:', error);
          const errorEl = document.getElementById('config-error');
          if (errorEl) {
            errorEl.textContent = '❌ Failed to export entries';
            errorEl.classList.remove('hidden');
            setTimeout(() => {
              errorEl.classList.add('hidden');
            }, 3000);
          }
        }
      });
    }
  }

  private async saveConfiguration(formData: FormData) {
    const errorEl = document.getElementById('config-error');
    const successEl = document.getElementById('config-success');
    
    if (errorEl) errorEl.classList.add('hidden');
    if (successEl) successEl.classList.add('hidden');

    const configData = {
      database_host: formData.get('database_host'),
      database_port: formData.get('database_port'),
      database_name: formData.get('database_name'),
      database_user: formData.get('database_user'),
      database_password: formData.get('database_password'),
      database_ssl_mode: formData.get('database_ssl_mode'),
      ollama_base_url: formData.get('ollama_base_url'),
      embedding_model: formData.get('embedding_model'),
      chat_model: formData.get('chat_model'),
    };

    try {
      const response = await fetch(`${API_BASE}/config`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(configData),
      });

      const data = await response.json();

      if (response.ok) {
        if (successEl) {
          successEl.innerHTML = `
            <strong>✅ Configuration saved!</strong><br>
            <br>
            Configuration reloaded successfully. Reloading page...
          `;
          successEl.classList.remove('hidden');
        }
        
        // Keep the modal open for 2 seconds, then reload the page
        setTimeout(() => {
          window.location.reload();
        }, 2000);
      } else {
        if (errorEl) {
          errorEl.textContent = data.error || 'Failed to save configuration';
          errorEl.classList.remove('hidden');
        }
      }
    } catch (error) {
      if (errorEl) {
        errorEl.textContent = 'Failed to connect to server';
        errorEl.classList.remove('hidden');
      }
    }
  }

  private async searchAttendees(query: string): Promise<string[]> {
    try {
      const response = await fetch(`${API_BASE}/attendees/search?q=${encodeURIComponent(query)}`);
      if (!response.ok) return [];
      const data = await response.json();
      return data.suggestions || [];
    } catch (error) {
      console.error('Failed to search attendees:', error);
      return [];
    }
  }

  private showAttendeesAutocomplete(input: HTMLInputElement, suggestions: string[]) {
    this.hideAttendeesAutocomplete();

    if (suggestions.length === 0) return;

    const autocomplete = document.createElement('div');
    autocomplete.className = 'attendees-autocomplete';
    
    suggestions.forEach(suggestion => {
      const item = document.createElement('div');
      item.className = 'autocomplete-item';
      item.textContent = suggestion;
      item.onmousedown = (e) => {
        // Use mousedown instead of click to fire before blur
        e.preventDefault();
        const currentValue = input.value;
        const lastComma = currentValue.lastIndexOf(',');
        
        if (lastComma >= 0) {
          // Replace the last name after comma
          input.value = currentValue.substring(0, lastComma + 1) + ' ' + suggestion + ', ';
        } else {
          // Replace entire value
          input.value = suggestion + ', ';
        }
        
        this.hideAttendeesAutocomplete();
        input.focus();
        this.scheduleAutoSave();
      };
      autocomplete.appendChild(item);
    });

    // Position it relative to the input's parent for better positioning
    const fieldContainer = input.closest('.form-field') as HTMLElement;
    if (fieldContainer) {
      fieldContainer.style.position = 'relative';
      autocomplete.style.position = 'absolute';
      autocomplete.style.left = '0';
      autocomplete.style.right = '0';
      autocomplete.style.top = `${input.offsetTop + input.offsetHeight}px`;
      fieldContainer.appendChild(autocomplete);
    } else {
      // Fallback to body positioning
      const rect = input.getBoundingClientRect();
      autocomplete.style.position = 'fixed';
      autocomplete.style.left = `${rect.left}px`;
      autocomplete.style.top = `${rect.bottom}px`;
      autocomplete.style.width = `${rect.width}px`;
      document.body.appendChild(autocomplete);
    }

    this.attendeesAutocomplete = autocomplete;
  }

  private hideAttendeesAutocomplete() {
    if (this.attendeesAutocomplete) {
      this.attendeesAutocomplete.remove();
      this.attendeesAutocomplete = null;
    }
  }

  private setupAttendeesAutocomplete(input: HTMLInputElement) {
    input.addEventListener('input', () => {
      if (this.autocompleteTimeout) {
        clearTimeout(this.autocompleteTimeout);
      }

      const value = input.value;
      // Get the text after the last comma (current name being typed)
      const lastComma = value.lastIndexOf(',');
      const currentName = lastComma >= 0 
        ? value.substring(lastComma + 1).trim() 
        : value.trim();

      if (currentName.length < 1) {
        this.hideAttendeesAutocomplete();
        return;
      }

      this.autocompleteTimeout = window.setTimeout(async () => {
        const suggestions = await this.searchAttendees(currentName);
        this.showAttendeesAutocomplete(input, suggestions);
      }, 200);
    });

    input.addEventListener('blur', () => {
      // Delay to allow clicking on autocomplete items
      setTimeout(() => this.hideAttendeesAutocomplete(), 200);
    });

    input.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        this.hideAttendeesAutocomplete();
      }
    });
  }
}

// Initialize app
new JournalApp();
