package businesscalendar

import (
	"webtyp.com/crud"
	"webtyp.com/model"
	"webtyp.com/storage"
)

func firstRow(tx crud.Tx, newRow func() model.Model, conds ...storage.Condition) (model.Model, bool, error) {
	var first model.Model
	err := tx.ReadAll(newRow, crud.Query{Conditions: conds, Limit: 1}, func(r model.Model) {
		if first == nil {
			first = r
		}
	})
	return first, first != nil, err
}

func readByID(tx crud.Tx, newRow func() model.Model, conds []storage.Condition, after string, limit int64, add func(model.Model) string) error {
	for {
		pageLimit := int64(crud.MaxPage)
		if limit > 0 && limit < pageLimit {
			pageLimit = limit
		}
		last, count := "", int64(0)
		err := tx.ReadAll(newRow, crud.Query{Conditions: conds, After: after, Limit: pageLimit}, func(r model.Model) {
			last = add(r)
			count++
		})
		if err != nil {
			return err
		}
		if limit > 0 {
			limit -= count
			if limit <= 0 {
				return nil
			}
		}
		if count < pageLimit {
			return nil
		}
		after = last
	}
}

func notFound(err error) error {
	if model.KindOf(err) == model.ErrorNotFound {
		return ErrNotFound
	}
	return err
}
