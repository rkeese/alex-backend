-- Migration: Seed SKR49 Booking Accounts
-- This migration populates the booking_accounts table with a standard set of SKR49 accounts.

INSERT INTO booking_accounts (club_id, majority_list, minority_list)
SELECT 
    id as club_id,
    m_list,
    min_list
FROM clubs
CROSS JOIN (
    VALUES 
    -- IDEELER BEREICH (Einnahmen)
    ('1_ideel', '2110 Mitgliedsbeiträge'),
    ('1_ideel', '2150 Aufnahmegebühren'),
    ('1_ideel', '2400 Spenden (steuerbegünstigt)'),
    ('1_ideel', '2301 Zuschüsse von Verbänden'),
    ('1_ideel', '2302 Zuschüsse öffentliche Hand'),
    ('1_ideel', '2303 Sonstige Zuschüsse'),
    ('1_ideel', '2400 Sonstige Einnahmen ideeller Bereich'),
    
    -- IDEELER BEREICH (Ausgaben)
    ('1_ideel', '2552 Ehrenamtspauschale'),
    ('1_ideel', '2554 Übungsleitervergütungen (Freibetrag)'),
    ('1_ideel', '2661 Miete, Nebenkosten (Geschäftsstelle)'),
    ('1_ideel', '2664 Reparaturen'),
    ('1_ideel', '2700 Kosten der Mitgliederverwaltung'),
    ('1_ideel', '2700 Kosten Kontoführung'),
    ('1_ideel', '2701 Büromaterial'),
    ('1_ideel', '2702 Porto, Telefon'),
    ('1_ideel', '2751 Abgaben Landesverband'),
    ('1_ideel', '2752 Abgaben Fachverband'),
    ('1_ideel', '2753 Versicherungen'),
    ('1_ideel', '2800 Mitgliederpflege'),
    ('1_ideel', '2802 Geschenke, Jubiläen, Ehrungen'),
    ('1_ideel', '2803 Ausbildungskosten'),
    ('1_ideel', '2804 Lehr- und Jugendarbeit'),
    ('1_ideel', '2894 Rechts- und Beratungskosten'),
    ('1_ideel', '2900 Sonstige Kosten'),

    -- VERMÖGENSVERWALTUNG (Einnahmen)
    ('2_vermoegen', '4150 Zinserträge'),
    ('2_vermoegen', '4151 Erträge aus Wertpapieren'),
    ('2_vermoegen', '4415 Mieten und Pachten (steuerfrei)'),
    
    -- VERMÖGENSVERWALTUNG (Ausgaben)
    ('2_vermoegen', '3000 Instandhaltung Immobilien'),
    
    -- ZWECKBETRIEB (Einnahmen)
    ('3_zweckbetrieb', '5005 Eintrittsgelder aus Wettkämpfen'),
    ('3_zweckbetrieb', '5070 Einnahmen aus sonstigen sportlichen Veranstaltungen'),
    ('3_zweckbetrieb', '5070 Schießeinnahmen'),
    ('3_zweckbetrieb', '5079 Startgelder'),
    ('3_zweckbetrieb', '5100 Einnahmen aus Leistungen gegenüber Mitgliedern'),
    ('3_zweckbetrieb', '5215 Zuschüsse von Behörden'),
    ('3_zweckbetrieb', '5225 Sonstige Zuschüsse'),
    ('3_zweckbetrieb', '5250 Einnahmen Munitionsverkauf'),
    ('3_zweckbetrieb', '5250 Sonstige Einnahmen Zweckbetrieb Sport'),

    -- ZWECKBETRIEB (Ausgaben)
    ('3_zweckbetrieb', '5518 Sonstige veranstaltungsabhängige Kosten'),
    ('3_zweckbetrieb', '5518 Startgelder'),
    ('3_zweckbetrieb', '5518 Ehrenscheiben, Pokale, Gravuren'),
    ('3_zweckbetrieb', '5545 Sonstige Kosten der Veranstaltung'),
    ('3_zweckbetrieb', '5570 Allgemeine Kosten des Sportbetriebs'),
    ('3_zweckbetrieb', '5570 Schießbedarf'),
    ('3_zweckbetrieb', '5570 Munition'),
    ('3_zweckbetrieb', '5575 Verwaltungskosten'),
    ('3_zweckbetrieb', '5575 Büromaterial'),
    ('3_zweckbetrieb', '5600 Versicherungen'),
    ('3_zweckbetrieb', '5605 Sportkleidung'),
    ('3_zweckbetrieb', '5609 Ausbildungskosten'),
    ('3_zweckbetrieb', '5610 Ausbildungskostenersatz'),
    ('3_zweckbetrieb', '5650 Sonstige Kosten Zweckbetrieb'),

    -- WIRTSCHAFTLICHER GESCHÄFTSBETRIEB (Einnahmen)
    ('4_wirtschaft', '8000 Umsatzerlöse Gastronomie'),
    ('4_wirtschaft', '8028 Erlöse Speisen/Getränke 7 % USt'),
    ('4_wirtschaft', '8131 Sonstige betriebliche Erträge'),

    -- WIRTSCHAFTLICHER GESCHÄFTSBETRIEB (Ausgaben)
    ('4_wirtschaft', '8150 Wareneinkauf Gastronomie'),
    ('4_wirtschaft', '8300 Anteilige Raumkosten'),
    ('4_wirtschaft', '8300 Vereinsheimkosten'),
    ('4_wirtschaft', '8303 Strom'),
    ('4_wirtschaft', '8304 Wasser'),
    ('4_wirtschaft', '8305 Brennstoff Heizung'),
    ('4_wirtschaft', '8310 Bürobedarf'),
    ('4_wirtschaft', '8318 Versicherungen, Beiträge'),
    ('4_wirtschaft', '8320 Sonstige Abgaben'),
    ('4_wirtschaft', '8320 Grundabgaben'),
    ('4_wirtschaft', '8320 Abfallentsorgung')
) AS t(m_list, min_list);
