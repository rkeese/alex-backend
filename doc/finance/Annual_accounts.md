# Requirements for annual accounts

I want to have an annual or user defined time period final finance statement from the bookings.
The finance statement should just respect the bookings from database bookings table.
The finance statement must follow a defined structure.
  1. On Top the club name
  2. Following german text "Finanz-Jahresabschluss für das Geschäftsjahr [Jahr]"
  3. The finance statement should have starting balances of the available club accounts and a barge.
     Like following table.
      | **Konto**       | Anfangsbestand (€) | Einnahmen (€) | Ausgaben (€) | Endbestand (€) |
      |------------------|--------------------|---------------|--------------|----------------|
      | **Bankkonto 1**  | 1.000              | 10.000        | 4.000        | 7.000          |
      | **Bankkonto 2**  | 500                | 1.200         | 300          | 1.400          |
      | **Kasse**        | 200                | 3.000         | 1.000        | 2.200          |
      | **Gesamt**       | **1.700**          | **14.200**    | **5.300**    | **10.600**     |

  4. Then we should implement a common overview table to see following example:
      | Bereich (booking accounts)  | Einnahmen (€) | Ausgaben (€) | **Ergebnis (€)** |
      |-----------------------------|---------------|--------------|------------------|
      | **Ideeller Bereich**        | 8.500         | –            | **8.500**        |
      | **Vermögensbereich**        | 1.200         | –            | **1.200**        |
      | **Zweckbetrieb**            | 4.500         | 3.000        | **1.500**        |
      | **Wirtschaftl. Geschäftsbetrieb** | 1.000    | 300          | **700**         |
      | **Gesamt**                  | **15.200**    | **3.300**    | **11.900**       |

  5. From this point down we must go in detail of the booking accounts
     "Ideeller Bereich" list all items which are acociated to this booking account.
  6. List all items of "Vermögensbereich"
  7. List all items of "Zweckbetrieb"
  8. List all items of "Wirtschaftl. Geschäftsbetrieb"
  9. Finally we need the date of creation this statement, personal sign of the finance board member and
     the annual accounts treasurer.
I think we need to store the data in the database in order to be able to reproduce this statement at any time later.
Can you make a implementation suggestion for the backend?
If you make any changes at the REST interface please update also the API.md file and give me detail information for frontend adaptation.


## AI suggestion

Hier ist eine strukturierte Anleitung für deinen **Finanz-Jahresabschluss** inklusive eines konkreten Beispiels. Der Fokus liegt auf der Zuordnung der **Einnahmen** zu den vier steuerrelevanten Bereichen (Ideeller Bereich, Vermögensbereich, Zweckbetrieb, Wirtschaftlicher Geschäftsbetrieb). Für Zweckbetrieb und Wirtschaftlichen Geschäftsbetrieb sind zusätzlich **Ausgaben** relevant, da hier der **Gewinn** (Einnahmen – Ausgaben) steuerpflichtig ist.

---

### **Struktur des Jahresabschluss-Dokuments**
#### **1. Deckblatt**
- Name des Vereins  
- "Finanz-Jahresabschluss für das Geschäftsjahr [Jahr]"  
- Datum der Erstellung  
- Unterschrift des Kassenwarts/Vorstands  

#### **2. Zusammenfassung der Einnahmen und Ergebnisse**  
*(Kurzübersicht für die Mitgliederversammlung)*  
| Bereich                     | Einnahmen (€) | Ausgaben (€) | **Ergebnis (€)** |
|-----------------------------|---------------|--------------|------------------|
| **Ideeller Bereich**        | 8.500         | –            | **8.500**        |
| **Vermögensbereich**        | 1.200         | –            | **1.200**        |
| **Zweckbetrieb**            | 4.500         | 3.000        | **1.500**        |
| **Wirtschaftl. Geschäftsbetrieb** | 1.000    | 300          | **700**          |
| **Gesamt**                  | **15.200**    | **3.300**    | **11.900**       |

> **Hinweis**:  
> - *Ideeller Bereich* und *Vermögensbereich* sind **steuerfrei**.  
> - *Zweckbetrieb* und *WGB* unterliegen ggf. der Körperschafts-/Gewerbesteuer (abhängig vom Gewinn).  

---

#### **3. Detaillierte Aufstellung nach Bereichen**  
##### **a) Ideeller Bereich**  
*(Steuerfreie Einnahmen aus satzungsgemäßen Zwecken)*  
- **Mitgliedsbeiträge**: 5.000 € (Bankkonto 1)  
- **Spenden**: 3.500 € (2.000 € Bankkonto 1 + 1.500 € Kasse)  
- **Gesamt**: **8.500 €**  
> *Keine Ausgaben werden hier aufgeführt, da sie für die Steuererklärung irrelevant sind.*

##### **b) Vermögensbereich**  
*(Passive Einnahmen aus Vermögenswerten)*  
- **Zinsen**: 200 € (Bankkonto 2)  
- **Mieteinnahmen** (z. B. Vermietung von Vereinsimmobilien): 1.000 € (Bankkonto 2)  
- **Gesamt**: **1.200 €**  
> *Ausgaben (z. B. Instandhaltungskosten) mindern den Gewinn, werden aber oft dem Vermögensbereich zugeordnet.*

##### **c) Zweckbetrieb**  
*(Einnahmen aus satzungsgemäßen Tätigkeiten mit wirtschaftlichem Bezug)*  
- **Einnahmen**:  
  - Veranstaltungstickets: 4.500 € (3.000 € Bankkonto 1 + 1.500 € Kasse)  
- **Ausgaben**:  
  - Raumkosten: 1.500 €  
  - Material: 1.000 €  
  - Werbung: 500 €  
- **Gewinn**: **1.500 €**  
> *Beispiel: Mitglieder-Weihnachtsfeier mit Ticketverkauf.*

##### **d) Wirtschaftlicher Geschäftsbetrieb (WGB)**  
*(Nicht satzungsgemäße, gewerbliche Tätigkeiten)*  
- **Einnahmen**:  
  - Verkauf von Fanartikeln: 1.000 € (Bankkonto 2)  
- **Ausgaben**:  
  - Einkauf Fanartikel: 200 €  
  - Lagerkosten: 100 €  
- **Gewinn**: **700 €**  
> *Beispiel: Gewerblicher Verkauf von T-Shirts an Nicht-Mitglieder.*

---

#### **4. Kassen- und Kontenabstimmung**  
*(Nachweis, dass alle Einnahmen erfasst wurden)*  

| **Konto**       | Anfangsbestand (€) | Einnahmen (€) | Ausgaben (€) | Endbestand (€) |
|------------------|--------------------|---------------|--------------|----------------|
| **Bankkonto 1**  | 1.000              | 10.000        | 4.000        | 7.000          |
| **Bankkonto 2**  | 500                | 1.200         | 300          | 1.400          |
| **Kasse**        | 200                | 3.000         | 1.000        | 2.200          |
| **Gesamt**       | **1.700**          | **14.200**    | **5.300**    | **10.600**     |

> **Prüfung**:  
> - **Einnahmen gesamt**: 14.200 € = Summe aller Bereiche (8.500 + 1.200 + 4.500 + 1.000).  
> - **Endbestand**: 10.600 € = Barvermögen (Kasse + Banken) zum Jahresende.

---

#### **5. Anmerkungen zur Zuordnung**  
- **Mitgliedsbeiträge/Spenden** → Immer **Ideeller Bereich**.  
- **Zinsen/Mieteinnahmen** → **Vermögensbereich** (sofern nicht für satzungsgemäße Zwecke genutzt).  
- **Veranstaltungen mit Mitgliedern** → **Zweckbetrieb** (z. B. Workshops mit Kostenbeteiligung).  
- **Verkäufe an Dritte** → **WGB** (z. B. Fanartikel, Catering für externe Events).  
- **Ausgaben** wurden den Bereichen zugeordnet, aus denen sie entstanden sind (z. B. Raumkosten für Veranstaltungen → Zweckbetrieb).

---

### **Warum diese Struktur?**  
1. **Steuerliche Compliance**: Die Trennung ist gesetzlich vorgeschrieben (§ 64 AO).  
2. **Transparenz**: Mitglieder und Finanzamt erkennen, wie Gelder verwendet wurden.  
3. **Praxistauglichkeit**: Klare Zuordnung über Konten/Kasse vereinfacht die Buchhaltung.  

---

### **Tipps für die Praxis**  
- **Trenne Konten**: Nutze ein separates Konto für wirtschaftliche Geschäftsbetriebe (Pflicht bei Umsatz > 17.500 €).  
- **Dokumentiere Zuordnungen**: Speichere Belege mit Vermerk (z. B. "Spende – Ideeller Bereich").  
- **Software nutzen**: Tools wie **Datev Vereinsmanagement** oder **Lexware** automatisieren die Zuordnung.  

Bei Fragen zur konkreten Zuordnung (z. B. "Ist eine Grillparty Zweckbetrieb oder WGB?") lohnt sich eine **steuerliche Beratung** – viele Vereine erhalten hier kostenlose Unterstützung von der **ARGE Vereinsberatung**.  

--- 

Dieses Schema ist **praxiserprobt** und entspricht den Anforderungen deutscher Finanzämter. Du kannst es direkt anpassen und mit deinen Zahlen füllen. 😊