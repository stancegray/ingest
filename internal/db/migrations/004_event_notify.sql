CREATE OR REPLACE FUNCTION notify_new_event()
RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify(
        'new_events',
        json_build_object('id', NEW.id)::text
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS events_notify ON events;
CREATE TRIGGER events_notify
    AFTER INSERT ON events
    FOR EACH ROW
    EXECUTE FUNCTION notify_new_event();
