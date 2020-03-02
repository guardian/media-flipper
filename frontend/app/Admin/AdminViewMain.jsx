import React from 'react';
import MenuBanner from "../MenuBanner.jsx";
import css from "./AdminView.css";
import rootcss from "../approot.css";

import QueueStatusWidget from "./QueueStatusWidget.jsx";

class AdminViewMain extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            showingDangerous: false,
            resultsList: [],
        }
    }

    async triggerPurge(queueName) {
        const url = "/api/jobrunner/purge?queue=" + queueName;
        const response = await fetch(url,{method: "DELETE"});
        if(response.status===200) {
            await response.body.cancel();
            this.setState(curState=>{return {resultsList: curState.resultsList.concat(["Purged " + queueName])}})
        } else {
            const errorBody =  await response.text();
            this.setState(curState=>{return {resultsList: curState.resultsList.concat(["Could not purge " + queueName + ": " + errorBody])}});
        }
    }

    render() {
        return <div>
            <MenuBanner/>
            <div className="admin-view-grid">
                <h1 onClick={()=>this.setState(oldState=>{return {showingDangerous: !oldState.showingDangerous}})}>Queue Status</h1>
                <QueueStatusWidget/>
            </div>
            {
                this.state.showingDangerous ? <div className="danger-area">
                    <h2 className="danger-header">Warning - dangerous actions for admins only</h2>
                    <div className="button-container">
                        <label className="button button-container-entry clickable" onClick={()=>this.triggerPurge("jobrunningqueue")}>Purge running queue</label>
                        <label className="button button-container-entry clickable" onClick={()=>this.triggerPurge("jobrequestqueue")}>Purge request queue</label>
                    </div>
                    <div className="results-area">
                        <ul className="admin-view-results-list">
                            {
                                this.state.resultsList.map((entry,idx)=><li key={idx}><pre>{entry}</pre></li>)
                            }
                        </ul>
                    </div>
                </div> : null
            }
        </div>
    }
}

export default AdminViewMain;